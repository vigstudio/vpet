"""Run against the isolated vpet-ui-test stack at localhost:18099.
Requires Python Playwright; no production URL or database mutation supported.
"""
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
      const response = await fetch(path, {
        method: body === null ? 'GET' : 'POST',
        headers: {'Content-Type':'application/json','X-CSRF-Token':csrf,
                  'X-Requested-With':'XMLHttpRequest'},
        ...(body === null ? {} : {body:JSON.stringify(body)})
      });
      if (!response.ok) throw new Error(path + ': ' + response.status);
      return response.json();
    }''', dict(path=path, body=body))


with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport=dict(width=1440, height=900), reduced_motion='reduce')
    errors = []
    page.on('pageerror', lambda error: errors.append(str(error)))
    try:
        page.goto(BASE)
        page.wait_for_load_state('networkidle')
        api(page, '/api/auth/register', dict(username='layout'+str(time.time_ns()),
            display_name='Layout test', password='Local-only-12345', locale='en'))
        api(page, '/api/v1/pet', dict(name='Layout pet', parts=PARTS))
        original = api(page, '/api/v1/state')
        original['welcome_gift'] = None
        original['halloween'] = False  # Event popup has a separate smoke test.
        for case in ['two', 'one', 'dead', 'care', 'empty', 'no-pet']:
            fixture = copy.deepcopy(original)
            fixture['slots'] = 1
            if case == 'two':
                fixture['pets'].append(dict(fixture['pets'][0], name='Second pet'))
                fixture['slots'] = 2
            if case == 'dead':
                fixture['pets'][0].update(dead=True, cause='test', died_at=fixture['now'])
            if case == 'care':
                fixture['pets'][0]['care_until'] = fixture['now'] + 3600000
            if case == 'empty':
                fixture['inv'] = {}
            if case == 'no-pet':
                fixture['pets'] = []
            page.route('**/api/v1/state', lambda route, request, data=fixture: route.fulfill(json=data))
            for width in [1440, 1024, 375]:
                page.set_viewport_size(dict(width=width, height=900))
                page.reload()
                page.wait_for_load_state('networkidle')
                gift = page.locator('.vpet-gift')
                if gift.count():
                    gift.get_by_role('button').last.click()
                close = page.locator('.vpet-checkin .vpet-iconbtn')
                if close.count():
                    close.first.click()
                page.locator('.vpet-tabs button').nth(1).click()
                assert len(page.locator('body').inner_text().strip()) > 60, case
                assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'), (case, width)
                assert page.locator('.vpet-switch').count() == (1 if case == 'two' else 0), case
                if case in ['two', 'one', 'care', 'empty']:
                    for section in ['hero', 'care', 'gear']:
                        assert page.locator('.vpet-sheet__'+section).evaluate("el => getComputedStyle(el).maxHeight === 'none'"), (case, section)
                if case == 'one' and width == 1440:
                    assert page.locator('.vpet-pettab').bounding_box()['width'] > 1000
                if case == 'dead':
                    assert page.locator('.vpet-dead').is_visible()
                if case == 'empty':
                    assert page.locator('.vpet-items__empty').is_visible()
                assert not errors, errors
            page.unroute('**/api/v1/state')
            print(case, 'OK', flush=True)
    finally:
        browser.close()
