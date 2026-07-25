const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const [,, command, testName, outputDir] = process.argv;
const BASE = 'http://127.0.0.1:4200';
const SHOT_DIR = outputDir || `/tmp/__test_recording/${testName}_shots`;
const VIDEO_PATH = `/tmp/__test_recording/${testName}_primary.mp4`;
const PID_FILE = '/tmp/dual_display_record.pid';

async function startRecording() {
    fs.mkdirSync(SHOT_DIR, { recursive: true });
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    const context = await browser.newContext({ viewport: { width: 1920, height: 1080 } });
    const page = await context.newPage();

    const pages = [
        '/login', '/dashboard', '/products', '/orders', '/customers',
        '/merchant/profile', '/merchant/settings', '/payouts',
        '/webhooks', '/providers', '/subscription', '/reports'
    ];

    for (let i = 0; i < pages.length; i++) {
        try {
            await page.goto(`${BASE}${pages[i]}`, { waitUntil: 'networkidle', timeout: 20000 });
            await page.waitForTimeout(2000); // Let Angular finish rendering
            await page.screenshot({ path: `${SHOT_DIR}/shot_${String(i+1).padStart(4, '0')}.png`, fullPage: false });
        } catch (e) {
            // Fallback: take screenshot even if timeout
            try { await page.screenshot({ path: `${SHOT_DIR}/shot_${String(i+1).padStart(4, '0')}.png` }); } catch(e2) {}
        }
    }
    await browser.close();

    // Stitch into video
    const shotGlob = `${SHOT_DIR}/shot_*.png`;
    execSync(`ffmpeg -y -framerate 0.5 -pattern_type glob -i "${shotGlob}" -c:v libx264 -preset ultrafast -crf 28 -pix_fmt yuv420p "${VIDEO_PATH}"`, { stdio: 'ignore' });
    fs.rmSync(PID_FILE);
    process.exit(0);
}

async function stopRecording() {
    if (fs.existsSync(PID_FILE)) {
        const pid = parseInt(fs.readFileSync(PID_FILE, 'utf8').trim());
        try { process.kill(pid, 'SIGTERM'); } catch(e) {}
        // Wait for video to be created
        for (let i = 0; i < 30; i++) {
            if (fs.existsSync(VIDEO_PATH)) break;
            await new Promise(r => setTimeout(r, 1000));
        }
        if (fs.existsSync(VIDEO_PATH)) {
            const size = fs.statSync(VIDEO_PATH).size;
            console.log(size);
        }
    }
    process.exit(0);
}

if (command === 'start') startRecording().catch(() => process.exit(1));
else if (command === 'stop') stopRecording().catch(() => process.exit(0));
else { console.log('Usage: node record.js {start|stop} <testName>'); process.exit(1); }
