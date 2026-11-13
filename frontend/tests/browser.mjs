// TODO(Tian): Expand browser coverage to additional MSE implementations and codecs.
import assert from 'node:assert/strict';
import { pathToFileURL } from 'node:url';

// Browser tooling is an optional development dependency, never a runtime one.
const modulePath = process.env.PLAYWRIGHT_MODULE;
if (!modulePath) throw new Error('Set PLAYWRIGHT_MODULE to the installed playwright/index.mjs path');
const { chromium } = await import(pathToFileURL(modulePath));
const browser = await chromium.launch({ headless: true, args: ['--autoplay-policy=no-user-gesture-required'] });
const page = await browser.newPage();
const errors = [];
page.on('pageerror', error => errors.push(String(error)));
try {
  await page.goto(process.argv[2]);
  await page.evaluate(() => {
    window.mediaEvents = [];
    for (const kind of ['loadstart', 'playing', 'waiting', 'seeking', 'seeked', 'ended', 'error']) {
      document.addEventListener(kind, event => {
        if (event.target instanceof HTMLVideoElement) window.mediaEvents.push({ kind, time: event.target.currentTime, source: event.target.currentSrc });
      }, true);
    }
  });
  let killedLeader = false;
  if (process.env.FAILOVER_LEADER_PID) {
    await page.route('**/api/v1/media/sessions/*/segments/1', async route => {
      if (!killedLeader) {
        await page.waitForFunction(() => document.querySelector('video')?.currentTime > 0.1);
        process.kill(Number(process.env.FAILOVER_LEADER_PID), 'SIGKILL');
        killedLeader = true;
      }
      await route.continue();
    });
  }
  await page.getByText('Advanced settings · Download and buffer', { exact: true }).click();
  await page.getByText('Experiment mode and manual video ID', { exact: true }).click();
  await page.getByLabel('Imported video ID').fill(process.env.MEDIA_VIDEO_ID || 'media-test');
  await page.getByLabel('Prefetch window (segments)').fill('2');
  await page.getByRole('button', { name: 'Apply and restart' }).click();
  await page.waitForFunction(() => document.querySelector('video')?.ended, null, { timeout: 60000 });
  const normal = await page.evaluate(() => ({ events: window.mediaEvents, error: document.querySelector('[role="alert"]')?.textContent,
    quality: { totalVideoFrames: document.querySelector('video').getVideoPlaybackQuality().totalVideoFrames }, source: document.querySelector('video').currentSrc }));
  assert.equal(normal.error, undefined, JSON.stringify(normal));
  assert.equal(normal.events.filter(event => event.kind === 'loadstart').length, 1, JSON.stringify(normal.events));
  const firstPlay = normal.events.findIndex(event => event.kind === 'playing');
  assert.ok(firstPlay >= 0, 'video never played');
  if (process.env.FAILOVER_LEADER_PID) assert.ok(killedLeader, 'leader failure was not injected');
  else assert.equal(normal.events.slice(firstPlay + 1).filter(event => event.kind === 'waiting').length, 0, JSON.stringify(normal.events));
  assert.ok(normal.quality.totalVideoFrames > 100, 'real frames were not decoded');
  // Seeking back after end exercises eviction, re-fetch, and MediaSource reopen.
  await page.evaluate(() => { const video = document.querySelector('video'); video.currentTime = 0.7; void video.play(); });
  await page.waitForFunction(() => { const v = document.querySelector('video'); return v.currentTime > 1 && !v.paused && !v.seeking; }, null, { timeout: 10000 });
  await page.evaluate(() => document.querySelector('video').pause());
  const paused = await page.evaluate(() => document.querySelector('video').currentTime);
  await page.waitForTimeout(350);
  assert.ok(Math.abs(await page.evaluate(() => document.querySelector('video').currentTime) - paused) < 0.05, 'pause advanced media time');
  await page.evaluate(() => { const video = document.querySelector('video'); video.currentTime = 6.5; void video.play(); });
  await page.waitForFunction(() => { const v = document.querySelector('video'); return v.currentTime > 6.8 && !v.seeking; }, null, { timeout: 10000 });
  const final = await page.evaluate(() => ({ events: window.mediaEvents, source: document.querySelector('video').currentSrc,
    error: document.querySelector('[role="alert"]')?.textContent }));
  assert.equal(final.source, normal.source, 'seek replaced the media source');
  assert.equal(final.events.filter(event => event.kind === 'loadstart').length, 1);
  assert.equal(final.error, undefined);
  assert.deepEqual(errors, []);
  await page.getByRole('button', { name: 'Stop playback', exact: true }).click();
  assert.equal(await page.locator('video').count(), 0);
  console.log(JSON.stringify({ status: 'PASS', browser: browser.version(), checks: ['one media source for all segments',
    process.env.FAILOVER_LEADER_PID ? 'automatic recovery after leader failure during playback' : 'no waiting at segment boundaries',
    'real video frame decode', 'seek after end with eviction', 'pause', 'forward seek', 'stop'],
    events: normal.events.map(({ kind, time }) => ({ kind, time })), frames: normal.quality.totalVideoFrames }));
} catch (error) {
  console.error(JSON.stringify({ browserErrors: errors, page: await page.locator('main').innerText(), events: await page.evaluate(() => window.mediaEvents) }));
  throw error;
} finally { await browser.close(); }
