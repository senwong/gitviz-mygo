async (page) => {
  const base = page.url().replace(/\/gitviz-mygo\/.*/, '/gitviz-mygo/');
  const routes = ['', 'features/', 'install/', 'comparison/', 'faq/', 'zh/'];
  const reports = [];
  for (const width of [1440, 390, 320]) {
    await page.setViewportSize({ width, height: 900 });
    for (const route of routes) {
      const response = await page.goto(base + route);
      if (!response || response.status() !== 200) throw new Error(`HTTP error: ${route}`);
      await page.locator('h1').waitFor();
      const result = await page.evaluate(() => ({
        width: innerWidth,
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        headings: document.querySelectorAll('h1').length,
        title: document.title,
        description: document.querySelector('meta[name="description"]')?.content,
        canonical: document.querySelector('link[rel="canonical"]')?.href,
        imageAltMissing: [...document.images].some(i => !i.hasAttribute('alt')),
        imageSizeMissing: [...document.images].some(i => !i.hasAttribute('width') || !i.hasAttribute('height')),
        gridColumns: document.querySelector('.feature-grid') ? getComputedStyle(document.querySelector('.feature-grid')).gridTemplateColumns : null,
        schema: JSON.parse(document.querySelector('script[type="application/ld+json"]').textContent),
      }));
      if (result.overflow || result.headings !== 1 || !result.description || !result.canonical || result.imageAltMissing || result.imageSizeMissing) {
        throw new Error(`Layout/metadata failed: ${route} ${JSON.stringify(result)}`);
      }
      if (width === 1440 && result.gridColumns && result.gridColumns.split(' ').length !== 3) {
        throw new Error(`Desktop feature grid is not three columns: ${result.gridColumns}`);
      }
      for (const image of await page.locator('img').all()) await image.scrollIntoViewIfNeeded();
      await page.evaluate(async () => { await Promise.all([...document.images].map(i => i.decode())); window.scrollTo(0, 0); });
      await page.screenshot({ path: `build/site-preview/${route.replaceAll('/', '') || 'home'}-${width}.png`, fullPage: true });
      reports.push({ route, width, title: result.title, overflow: result.overflow, gridColumns: result.gridColumns });
    }
  }
  await page.setViewportSize({ width: 720, height: 900 });
  await page.goto(base);
  await page.evaluate(() => { document.documentElement.style.fontSize = '200%'; });
  const zoomOverflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
  if (zoomOverflow) throw new Error('Page overflows with 200% text size');
  await page.screenshot({ path: 'build/site-preview/home-text-200.png', fullPage: true });
  await page.goto(base);
  await page.keyboard.press('Tab');
  if (await page.locator(':focus').textContent() !== 'Skip to content') throw new Error('Skip link is not first keyboard target');
  await page.keyboard.press('Enter');
  if (!page.url().endsWith('#main')) throw new Error('Skip link does not reach main content');
  await page.getByRole('link', { name: /Features/, exact: true }).first().click();
  if (!page.url().endsWith('/features/')) throw new Error('Feature navigation failed');
  await page.getByRole('link', { name: 'Install', exact: true }).click();
  if (!page.url().endsWith('/install/')) throw new Error('Installation navigation failed');
  return { pages: reports.length, reports, textZoom200: 'pass', keyboardSkipLink: 'pass', navigation: 'pass' };
}
