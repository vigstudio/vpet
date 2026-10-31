"""Local-only regression: notifications must never steal navigation or open PvP."""
import copy
import time
from playwright.sync_api import sync_playwright

BASE = 'http://localhost:18099'
PARTS = dict(head=0, top=2, eyes=0, mouth=0, body=0, pattern=0,
             legs=0, tail=1, c1=3, c2=1, c3=2)


def api(page, path, body=None):
    return page.evaluate('''async ({path,body}) => {
      const csrf = document.cookie.split(';').map(c=>c.trim())
        .find(c=>c.startsWith('vpet_csrf='))?.slice(10) || '';
      const r = await fetch(path, {method:body===null?'GET':'POST',
        headers:{'Content-Type':'application/json','X-CSRF-Token':csrf,'X-Requested-With':'XMLHttpRequest'},
        ...(body===null?{}:{body:JSON.stringify(body)})});
      if (!r.ok) throw new Error(path+': '+r.status);
      return r.json();
    }''', dict(path=path, body=body))


with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    try:
        for width, height in [(375, 740), (1440, 900)]:
            page = browser.new_page(viewport=dict(width=width, height=height), reduced_motion='reduce')
            errors = []
            page.on('pageerror', lambda e: errors.append(str(e)))
            page.add_init_script('''
              window.streams = [];
              window.EventSource = class extends EventTarget {
                static CLOSED = 2;
                constructor(url) { super(); this.url=url; this.readyState=1; streams.push(this); }
                close() { this.readyState=2; }
              };
            ''')
            page.goto(BASE)
            page.wait_for_load_state('networkidle')
            api(page, '/api/auth/register', dict(username='notify'+str(time.time_ns()),
                password='Local-only-12345', locale='en'))
            api(page, '/api/v1/pet', dict(name='My pet', parts=PARTS))
            fixture = copy.deepcopy(api(page, '/api/v1/state'))
            fixture.update(welcome_gift=None, checkin=None, halloween=False)
            pet = fixture['pets'][0]
            fixture['duels'] = [dict(id='old-duel', at=fixture['now'], winner=0,
                a=dict(name='My pet', user='me', born=pet['born'], parts=PARTS, hp=100),
                b=dict(name='Opponent', user='other', born=22, parts=PARTS, hp=100),
                turns=[dict(by=0, hp=[100, 0], text='Finished')])]
            requests = []

            def state_response(route):
                requests.append(1)
                route.fulfill(json=fixture)

            page.route('**/api/v1/state', state_response)
            for tab in ['market', 'city']:
                page.evaluate('(tab) => localStorage.setItem("vpet.tab",tab)', tab)
                page.reload()
                page.wait_for_load_state('networkidle')
                page.locator('.vpet-world-shell.is-screen-'+tab).wait_for()
                before = len(requests)
                page.evaluate('''() => streams[0].onmessage({data:JSON.stringify({
                  id:99, tab:'duel', duel:'old-duel', message:'PvP notification regression'
                })})''')
                page.get_by_text('PvP notification regression', exact=True).wait_for()
                page.wait_for_timeout(300)
                assert len(requests) > before, 'notification stopped refreshing game state'
                assert page.locator('.vpet-world-shell.is-screen-'+tab).count(), ('notification stole tab', width, tab)
                assert not page.locator('.vpet-arena').count(), 'old notification auto-opened replay'
                assert page.evaluate('localStorage.getItem("vpet.tab")') == tab
                # Exercise the manual reconnect branch, then replay the same old notification.
                page.evaluate('''() => { streams[0].readyState=EventSource.CLOSED; streams[0].onerror(); }''')
                page.wait_for_function('() => streams.length === 2')
                page.evaluate('''() => streams[1].onmessage({data:JSON.stringify({
                  id:99, tab:'duel', duel:'old-duel', message:'Reconnected PvP notification'
                })})''')
                page.get_by_text('Reconnected PvP notification', exact=True).wait_for()
                assert page.locator('.vpet-world-shell.is-screen-'+tab).count()
                assert not page.locator('.vpet-arena').count()
            page.locator('.vpet-tabs > button').nth(9).dispatch_event('click')
            page.locator('.vpet-duel__tabs button').last.click()
            page.get_by_role('button', name='▶ Xem lại', exact=True).click()
            page.locator('.vpet-arena').wait_for()
            assert not errors, errors
            print(width, 'reload/reconnect preserve tabs, toast/refresh and manual PvP replay: OK', flush=True)
            page.close()
    finally:
        browser.close()
