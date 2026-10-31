"""Local-only UI regression; requires isolated stack at localhost:18099."""
import copy
import os
import tempfile
import time
from playwright.sync_api import sync_playwright

BASE = 'http://localhost:18099'
PARTS = dict(head=0, top=2, eyes=0, mouth=0, body=0, pattern=0,
             legs=0, tail=1, c1=3, c2=1, c3=2)


def api(page, path, body=None):
    return page.evaluate('''async ({path,body}) => {
      const csrf = document.cookie.split(';').map(c=>c.trim())
        .find(c=>c.startsWith('vpet_csrf='))?.slice(10) || '';
      const response = await fetch(path, {method:body===null?'GET':'POST',
        headers:{'Content-Type':'application/json','X-CSRF-Token':csrf,'X-Requested-With':'XMLHttpRequest'},
        ...(body===null?{}:{body:JSON.stringify(body)})});
      if (!response.ok) throw new Error(path+': '+response.status);
      return response.json();
    }''', dict(path=path, body=body))


with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport=dict(width=375, height=740), reduced_motion='reduce')
    errors = []
    page.on('pageerror', lambda error: errors.append(str(error)))
    try:
        page.goto(BASE)
        page.wait_for_load_state('networkidle')
        api(page, '/api/auth/register', dict(username='event'+str(time.time_ns()),
            password='Local-only-12345', locale='en'))
        api(page, '/api/v1/pet', dict(name='Living pet', parts=PARTS))
        original = api(page, '/api/v1/state')
        fixture = copy.deepcopy(original)
        fixture.update(welcome_gift=None, checkin=None, halloween=True, slots=2)
        fixture['pets'].append(dict(fixture['pets'][0], name='Second pet', born=22))
        snapshot = dict(fixture['pets'][0], name='Lost companion', born=11, lv=12)
        fixture['graves'] = [dict(name=snapshot['name'], parts=PARTS, born=11,
                                 died=fixture['now']-1000, cause='Test', snapshot=snapshot)]
        page.route('**/api/v1/state', lambda route: route.fulfill(json=fixture))
        page.reload()
        page.wait_for_load_state('networkidle')
        popup = page.locator('.vpet-halloween')
        popup.wait_for(state='visible')
        assert popup.evaluate('el => el.open && el.contains(document.activeElement)')
        page.keyboard.press('Escape')
        popup.wait_for(state='detached')
        page.reload()
        page.wait_for_load_state('networkidle')
        assert not popup.count(), 'event advertisement repeated after dismissal'
        page.evaluate("Object.keys(localStorage).filter(k=>k.startsWith('vpet.halloween.')).forEach(k=>localStorage.removeItem(k))")
        page.reload()
        page.wait_for_load_state('networkidle')
        page.get_by_role('button', name='Visit the graveyard', exact=True).click()
        page.get_by_role('button', name='Choose revival').click()
        confirm = page.locator('.vpet-graveyard .vpet-market__confirm')
        assert confirm.is_visible()
        revive = page.get_by_role('button', name='Sacrifice and revive')
        assert revive.is_disabled()
        confirm.locator('select').select_option('22')
        assert revive.is_disabled()
        confirm.locator('input[type=checkbox]').check()
        assert revive.is_enabled()
        page.get_by_role('button', name='Settings', exact=True).click()
        audio = page.locator('.vpet-audio')
        audio.locator('input[type=checkbox]').first.check()
        audio.locator('input[type=checkbox]').last.check()
        audio.locator('input[type=range]').first.fill('0.25')
        assert page.evaluate("JSON.parse(localStorage.getItem('vpet.audio')).music")
        audio.locator('input[type=checkbox]').first.uncheck()
        audio.locator('input[type=checkbox]').last.uncheck()
        page.keyboard.press('Escape')
        page.route('**/api/v1/mute', lambda route: route.fulfill(json=dict(fixture, muted=True, message='Mobile notice: two pets, active event and successful action.')))
        page.get_by_role('button', name='Settings', exact=True).click()
        page.locator('.vpet-mute input').click()
        page.locator('.vpet-toast').wait_for(state='visible')
        page.keyboard.press('Escape')
        for width, height in [(360,640),(375,740),(390,844),(768,1024),(1440,900),(740,375),(844,390)]:
            page.set_viewport_size(dict(width=width, height=height))
            page.locator('.vpet-tabs > button').nth(1).click()
            if width <= 760:
                sections = page.locator('.vpet-pettab > .vpet-phone-sections button')
                assert sections.count() == 3
                for button in sections.all():
                    button.click()
                    assert button.bounding_box()['height'] >= 44
                sections.first.click()
            assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'), (width, height)
            header = page.locator('.vpet-panel__head').bounding_box()
            if width <= 760:
                assert header['height'] <= 75, (width, height, header)
            if height > 500:
                page.screenshot(path=os.path.join(tempfile.gettempdir(), f'vpet-mobile-{width}.png'))
            page.locator('.vpet-tabs > button').nth(3).click()
            page.evaluate('window.scrollTo(0,0)')
            if height < 500:
                cta = page.locator('.vpet-campaign__controls .vpet-expedition-bar button').first
                box = cta.bounding_box()
                nav = page.locator('.vpet-tabs').bounding_box()
                print('landscape CTA', width, height, box, 'nav', nav, flush=True)
                page.screenshot(path=os.path.join(tempfile.gettempdir(), f'vpet-landscape-{width}.png'))
                assert box['y'] >= 0 and box['y']+box['height'] <= nav['y'], (width, height, box, nav)
            page.locator('.vpet-tabs > button').nth(8).dispatch_event('click')
            if width <= 760:
                market_sections = page.locator('.vpet-market > .vpet-phone-sections button')
                assert market_sections.count() == 3
                for index, button in enumerate(market_sections.all()):
                    button.click()
                    assert button.bounding_box()['height'] >= 44
                    assert page.locator('.vpet-market__board').is_visible() == (index == 0)
                    assert page.locator('.vpet-market__stall').is_visible() == (index != 0)
            assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'), ('market', width)
            print(width, height, 'OK', flush=True)
        assert not errors, errors
        print('Popup, sacrifice confirmation, pet sections, market sections, layouts: OK', flush=True)
    finally:
        browser.close()
