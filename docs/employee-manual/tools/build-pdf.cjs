#!/usr/bin/env node
/*
 * Build the BeaconChip employee manual PDF from ../README.md.
 *
 * Usage (from any directory that has these packages installed):
 *   npm i marked@14 playwright-core pdfjs-dist@4 pdf-lib \
 *     @fontsource/noto-sans-sc @fontsource/noto-serif-sc @fontsource/jetbrains-mono
 *   NODE_PATH=$PWD/node_modules CHROME_PATH=/path/to/chrome \
 *     node docs/employee-manual/tools/build-pdf.cjs
 *
 * Output: docs/employee-manual/BeaconChip-员工使用手册.pdf
 *
 * The PDF is rendered twice: the first pass locates the page each heading
 * lands on, the second pass writes those page numbers into the contents page.
 */
const fs = require('fs')
const path = require('path')
const { pathToFileURL } = require('url')
const { marked } = require('marked')
const { chromium } = require('playwright-core')

const ROOT = path.resolve(__dirname, '..')
const SRC = path.join(ROOT, 'README.md')
const OUT = path.join(ROOT, 'BeaconChip-员工使用手册.pdf')
const LOGO = path.resolve(ROOT, '../../frontend/public/logo.svg')
const CHROME = process.env.CHROME_PATH || '/opt/pw-browsers/chromium-1194/chrome-linux/chrome'
const VERSION = process.env.MANUAL_VERSION || 'v1.1'
const EDITION = process.env.MANUAL_EDITION || '2026 年 9 月'

const fontCss = (pkg, weights) =>
  weights.map((w) => fs.readFileSync(require.resolve(`@fontsource/${pkg}/${w}.css`), 'utf8')
    .replace(/url\(\.\/files\//g, `url(${pathToFileURL(path.dirname(require.resolve(`@fontsource/${pkg}/${w}.css`))).href}/files/`)).join('\n')

const slug = (s) => s.replace(/<[^>]+>/g, '').trim()
const norm = (s) => s.replace(/\s+/g, '')

function parseManual() {
  let md = fs.readFileSync(SRC, 'utf8')
  // The title block and the markdown contents list are replaced by the cover and generated contents page.
  const firstChapter = md.indexOf('\n## 1. ')
  const intro = md.slice(0, firstChapter)
  md = md.slice(firstChapter)
  const introQuote = (intro.match(/^> (?!\*\*)(.+)$/m) || [])[1] || ''
  const noteQuote = (intro.match(/^> 说明：(.+)$/m) || [])[1] || ''
  const infoTable = (intro.match(/\| 项目 \| 值 \|[\s\S]*?\n\n/) || [''])[0]
  return { md, introQuote, noteQuote, infoTable }
}

function renderBody(md) {
  const chapters = []
  let current = null
  const renderer = new marked.Renderer()
  renderer.heading = function ({ tokens, depth }) {
    const text = this.parser.parseInline(tokens)
    if (depth === 2) {
      const m = slug(text).match(/^(\d+)\.\s*(.+)$/)
      const num = m ? m[1].padStart(2, '0') : ''
      const title = m ? m[2] : slug(text)
      current = { id: `ch-${num}`, num, title, subs: [] }
      chapters.push(current)
      return `<section class="chapter-head" id="${current.id}">
        <div class="chapter-kicker">CHAPTER ${num}</div>
        <h2><span class="chapter-num">${num}</span>${title}</h2>
      </section>`
    }
    if (depth === 3 && current) {
      const id = `${current.id}-s${current.subs.length + 1}`
      current.subs.push({ id, title: slug(text) })
      return `<h3 id="${id}">${text}</h3>`
    }
    return `<h${depth}>${text}</h${depth}>`
  }
  renderer.image = ({ href, text }) =>
    `<figure><img src="${pathToFileURL(path.join(ROOT, href)).href}" alt="${text}"/><figcaption>图 · ${text}</figcaption></figure>`
  renderer.blockquote = function ({ tokens }) {
    const inner = this.parser.parse(tokens)
    const plain = inner.replace(/<[^>]+>/g, '')
    let cls = 'note'
    if (plain.includes('⭐')) cls = 'star'
    else if (plain.includes('⚠️')) cls = 'warn'
    else if (plain.includes('💡')) cls = 'tip'
    return `<aside class="callout ${cls}">${inner}</aside>`
  }
  renderer.code = ({ text, lang }) => {
    const esc = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    const label = { bash: 'Terminal · macOS / Linux', bat: 'Windows CMD', powershell: 'PowerShell', json: 'JSON', toml: 'TOML' }[lang] || lang || ''
    return `<div class="code"><div class="code-bar"><i></i><i></i><i></i><span>${label}</span></div><pre><code>${esc}</code></pre></div>`
  }
  renderer.hr = () => ''
  const html = marked.parse(md, { renderer, gfm: true })
  return { html, chapters }
}

function buildHtml({ body, chapters, pages, introQuote, noteQuote, infoTable }, part) {
  const logo = fs.readFileSync(LOGO, 'utf8')
  const pg = (id) => (pages && pages[id] ? String(pages[id]) : '00')
  const toc = chapters.map((c) => `
    <li class="toc-ch"><a href="#${c.id}"><span class="toc-num">${c.num}</span><span class="toc-title">${c.title}</span><span class="toc-dots"></span><span class="toc-pg">${pg(c.id)}</span></a>
      ${c.subs.length ? `<ol>${c.subs.map((s) => `<li><a href="#${s.id}"><span class="toc-title">${s.title}</span><span class="toc-dots"></span><span class="toc-pg">${pg(s.id)}</span></a></li>`).join('')}</ol>` : ''}
    </li>`).join('')
  const info = marked.parse(infoTable || '')

  return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>BeaconChip 员工使用手册</title>
<style>
${fontCss('noto-sans-sc', ['400', '500', '700', '900'])}
${fontCss('noto-serif-sc', ['700'])}
${fontCss('jetbrains-mono', ['400', '600'])}
:root{--ink:#0f172a;--muted:#475569;--line:#e2e8f0;--brand:#4f46e5;--brand2:#06b6d4;--navy:#0a1a39;--soft:#f5f7fb}
@page{size:A4;margin:22mm 18mm 20mm 18mm}
@page cover{margin:0}
*{box-sizing:border-box}
html{-webkit-print-color-adjust:exact;print-color-adjust:exact}
body{margin:0;font-family:'Noto Sans SC','JetBrains Mono',sans-serif;color:var(--ink);font-size:10.2pt;line-height:1.75;background:#fff}
a{color:var(--brand);text-decoration:none}
strong{font-weight:700;color:#111827}
p{margin:.45em 0 .7em}
ul,ol{padding-left:1.4em;margin:.4em 0 .8em}
li{margin:.18em 0}

/* ---------- cover ---------- */
.cover{page:cover;height:297mm;width:210mm;position:relative;overflow:hidden;color:#fff;
  background:radial-gradient(1200px 700px at 85% -10%,rgba(6,182,212,.35),transparent 60%),radial-gradient(900px 600px at -10% 110%,rgba(79,70,229,.45),transparent 60%),linear-gradient(160deg,#142b56 0%,#0a1a39 55%,#061127 100%);page-break-after:always}
.cover .grid{position:absolute;inset:0;background-image:linear-gradient(rgba(255,255,255,.05) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.05) 1px,transparent 1px);background-size:14mm 14mm}
.cover .inner{position:absolute;left:22mm;right:22mm;top:30mm;bottom:24mm;display:flex;flex-direction:column}
.cover .logo{width:30mm;height:30mm;border-radius:7mm;overflow:hidden;box-shadow:0 10px 40px rgba(0,0,0,.45)}
.cover .logo svg{width:100%;height:100%;display:block}
.cover .brand{margin-top:14mm;font-size:46pt;font-weight:900;letter-spacing:-.5pt;line-height:1.05}
.cover .tag{font-size:14pt;color:#a5b4fc;letter-spacing:3pt;margin-top:3mm;font-weight:500}
.cover .title{margin-top:22mm;font-family:'Noto Serif SC',serif;font-size:34pt;font-weight:700;line-height:1.25}
.cover .title small{display:block;font-family:'Noto Sans SC';font-size:12pt;font-weight:400;color:#cbd5e1;margin-top:5mm;letter-spacing:.5pt;max-width:140mm;line-height:1.7}
.cover .bar{width:24mm;height:1.6mm;border-radius:1mm;background:linear-gradient(90deg,var(--brand2),var(--brand));margin-top:12mm}
.cover .meta{margin-top:auto;display:grid;grid-template-columns:1.7fr 1fr 1fr;gap:6mm;border-top:1px solid rgba(255,255,255,.18);padding-top:7mm}
.cover .meta div{font-size:8.5pt;color:#94a3b8;letter-spacing:1pt}
.cover .meta b{display:block;font-size:11pt;color:#fff;font-weight:500;letter-spacing:0;margin-top:1.5mm;font-family:'JetBrains Mono','Noto Sans SC';white-space:nowrap}
.cover .stamp{position:absolute;right:22mm;top:30mm;border:1px solid rgba(255,255,255,.35);border-radius:99px;padding:1.5mm 5mm;font-size:9pt;letter-spacing:2pt;color:#e2e8f0}

/* ---------- front matter ---------- */
.front{page-break-after:always}
.eyebrow{font-size:9pt;letter-spacing:3pt;color:var(--brand);font-weight:700}
.front h1{font-family:'Noto Serif SC',serif;font-size:24pt;margin:2mm 0 6mm}
.lead{font-size:11pt;color:var(--muted);line-height:1.9}
.front table{margin-top:6mm}
.toc{list-style:none;padding:0;margin:4mm 0 0}
.toc a{display:flex;align-items:baseline;color:var(--ink)}
.toc-ch{margin:0 0 2.2mm}
.toc-ch>a{font-weight:700;font-size:11pt;padding:2mm 0;border-bottom:1px solid var(--line)}
.toc-num{font-family:'JetBrains Mono';color:var(--brand);width:11mm;flex:none;font-weight:600}
.toc-dots{flex:1;border-bottom:1px dotted #cbd5e1;margin:0 3mm;transform:translateY(-1.2mm)}
.toc-pg{font-family:'JetBrains Mono';font-size:9.5pt;color:var(--muted)}
.toc-ch ol{list-style:none;padding:0 0 0 11mm;margin:1mm 0 0}
.toc-ch ol a{font-size:9.5pt;padding:.6mm 0;color:#334155}

/* ---------- chapters ---------- */
.chapter-head{page-break-before:always;margin:0 0 7mm;padding:0 0 5mm;border-bottom:2px solid var(--ink);position:relative}
.chapter-kicker{font-family:'JetBrains Mono';font-size:8.5pt;letter-spacing:3pt;color:var(--brand);font-weight:600}
.chapter-head h2{font-family:'Noto Serif SC',serif;font-size:22pt;margin:2mm 0 0;display:flex;align-items:baseline;gap:4mm;line-height:1.3}
.chapter-num{font-family:'JetBrains Mono';font-size:30pt;font-weight:600;color:transparent;-webkit-text-stroke:1px #94a3b8}
h3{font-size:13pt;margin:8mm 0 3mm;padding-left:3.5mm;border-left:3px solid var(--brand);line-height:1.4;break-after:avoid}
h4{font-size:11pt;margin:5mm 0 2mm}
h3+p,h3+figure{break-before:avoid}
p:has(+ figure),p:has(+ .code),p:has(+ ul),h4{break-after:avoid}
li:has(figure){break-inside:avoid}

table{width:100%;border-collapse:separate;border-spacing:0;margin:3mm 0 5mm;font-size:9pt;border:1px solid var(--line);border-radius:2.5mm;overflow:hidden;break-inside:auto}
thead th{background:#eef2ff;color:#312e81;text-align:left;font-weight:700;padding:2.2mm 3mm;border-bottom:1px solid #c7d2fe}
td{padding:2mm 3mm;border-bottom:1px solid var(--line);vertical-align:top}
td:first-child{min-width:24mm}
tr:last-child td{border-bottom:0}
tbody tr:nth-child(even) td{background:#fafbff}
tr{break-inside:avoid}

code{font-family:'JetBrains Mono','Noto Sans SC',monospace;font-size:.86em;background:#eef2ff;color:#3730a3;padding:.15em .4em;border-radius:1mm}
.code{margin:3mm 0 5mm;border-radius:2.5mm;overflow:hidden;background:#0b1220;break-inside:avoid;box-shadow:0 2px 0 rgba(15,23,42,.04)}
.code-bar{display:flex;align-items:center;gap:1.4mm;padding:1.8mm 3.5mm;background:#1e293b}
.code-bar i{width:2.2mm;height:2.2mm;border-radius:50%;background:#ef4444;display:block}
.code-bar i:nth-child(2){background:#f59e0b}.code-bar i:nth-child(3){background:#22c55e}
.code-bar span{margin-left:2.5mm;font-family:'JetBrains Mono';font-size:7.5pt;color:#94a3b8;letter-spacing:.5pt}
.code pre{margin:0;padding:3.5mm 4mm;white-space:pre-wrap;word-break:break-all}
.code pre code{background:none;color:#e2e8f0;padding:0;font-size:8.4pt;line-height:1.65}

figure{margin:4mm 0 6mm;break-inside:avoid;text-align:center}
figure img{max-width:100%;max-height:118mm;border:1px solid var(--line);border-radius:2.5mm;box-shadow:0 6px 18px rgba(15,23,42,.10)}
figcaption{font-size:8.5pt;color:#64748b;margin-top:2mm}
li figure{margin-left:-1.4em}

.callout{margin:4mm 0 5mm;padding:3.5mm 4.5mm;border-radius:2.5mm;border:1px solid;break-inside:avoid;font-size:9.8pt}
.callout p{margin:.2em 0}
.callout.note{background:#f8fafc;border-color:#e2e8f0;color:#334155}
.callout.tip{background:#ecfeff;border-color:#a5f3fc;color:#155e75}
.callout.warn{background:#fff7ed;border-color:#fed7aa;color:#9a3412}
.callout.star{background:linear-gradient(135deg,#eef2ff,#ecfeff);border-color:#a5b4fc;color:#1e1b4b;border-left:4px solid var(--brand);padding:4.5mm 5mm;font-size:10.4pt}
.callout.star strong{color:#3730a3}
</style></head><body>

${part === 'cover' ? `<section class="cover"><div class="grid"></div>
  <div class="stamp">INTERNAL · 内部资料</div>
  <div class="inner">
    <div class="logo">${logo}</div>
    <div class="brand">BeaconChip</div>
    <div class="tag">API GATEWAY PLATFORM</div>
    <div class="bar"></div>
    <div class="title">员工使用手册<small>${introQuote.replace(/\*\*/g, '')}</small></div>
    <div class="meta">
      <div>控制台 / API 端点<b>api.beaconchip-token.com</b></div>
      <div>版本<b>${VERSION}</b></div>
      <div>发布日期<b>${EDITION}</b></div>
    </div>
  </div>
</section>` : `
<section class="front">
  <div class="eyebrow">ABOUT</div>
  <h1>关于本手册</h1>
  <p class="lead">本手册面向使用 BeaconChip 平台的全体员工，介绍从登录、创建 API 密钥、在 Claude Code / Codex 等工具中接入，到查看用量、账号安全与故障排查的完整流程。</p>
  ${info}
  <aside class="callout star"><p>⭐ <strong>最快上手路径：</strong>安装 <strong>CC Switch</strong> → 在“API 密钥”页点击 <strong>导入到 CCS</strong> → 在 CC Switch 中确认启用。无需打开终端、无需手动编辑配置文件，详见第 5.1 节。</p></aside>
  <aside class="callout note"><p>${noteQuote}</p></aside>
</section>

<section class="front">
  <div class="eyebrow">CONTENTS</div>
  <h1>目录</h1>
  <ol class="toc">${toc}</ol>
</section>

<main>${body}</main>`}
</body></html>`
}

const footer = `<div style="width:100%;font-size:7.5pt;color:#94a3b8;padding:0 18mm;display:flex;justify-content:space-between;font-family:'WenQuanYi Zen Hei',sans-serif">
  <span>BeaconChip 员工使用手册 · 内部资料</span><span><span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`
const header = `<div style="width:100%;font-size:7pt;color:#cbd5e1;padding:0 18mm;text-align:right;letter-spacing:2px;font-family:'WenQuanYi Zen Hei',sans-serif">BEACONCHIP · API GATEWAY PLATFORM</div>`

async function render(browser, html, file, chrome = true) {
  const tmp = path.join(ROOT, '.manual-print.html')
  fs.writeFileSync(tmp, html)
  const page = await browser.newPage()
  await page.goto(pathToFileURL(tmp).href, { waitUntil: 'networkidle' })
  await page.evaluate(() => document.fonts.ready)
  await page.pdf({ path: file, format: 'A4', printBackground: true, preferCSSPageSize: true, displayHeaderFooter: chrome, headerTemplate: header, footerTemplate: footer, outline: true, tagged: true })
  await page.close()
  fs.unlinkSync(tmp)
}

async function locatePages(file, chapters) {
  const pdfjs = await import(pathToFileURL(require.resolve('pdfjs-dist/legacy/build/pdf.mjs')).href)
  const doc = await pdfjs.getDocument({ data: new Uint8Array(fs.readFileSync(file)), verbosity: 0 }).promise
  const texts = []
  for (let i = 1; i <= doc.numPages; i++) {
    const tc = await (await doc.getPage(i)).getTextContent()
    texts.push(norm(tc.items.map((it) => it.str).join('')))
  }
  const pages = {}
  let from = 2 // skip the about and contents pages
  for (const c of chapters) {
    const idx = texts.findIndex((t, i) => i >= from && t.includes(`CHAPTER${c.num}`))
    if (idx >= 0) { pages[c.id] = idx + 1; from = idx }
    for (const s of c.subs) {
      const j = texts.findIndex((t, i) => i >= from && t.includes(norm(s.title)))
      if (j >= 0) pages[s.id] = j + 1
    }
  }
  return { pages, total: doc.numPages }
}

;(async () => {
  const parsed = parseManual()
  const { html: body, chapters } = renderBody(parsed.md)
  const coverFile = OUT + '.cover.pdf'
  const bodyFile = OUT + '.body.pdf'
  const browser = await chromium.launch({ executablePath: CHROME })
  await render(browser, buildHtml({ ...parsed, body, chapters }, 'cover'), coverFile, false)
  await render(browser, buildHtml({ ...parsed, body, chapters }, 'body'), bodyFile)
  // Page numbers in the footer and contents count from the first page after the cover.
  const { pages } = await locatePages(bodyFile, chapters)
  await render(browser, buildHtml({ ...parsed, body, chapters, pages }, 'body'), bodyFile)
  await browser.close()

  // Insert the cover into the body document so its outline and internal links stay intact.
  const { PDFDocument } = require('pdf-lib')
  const out = await PDFDocument.load(fs.readFileSync(bodyFile))
  const cover = await PDFDocument.load(fs.readFileSync(coverFile))
  const [coverPage] = await out.copyPages(cover, [0])
  out.insertPage(0, coverPage)
  fs.unlinkSync(coverFile)
  fs.unlinkSync(bodyFile)
  out.setTitle('BeaconChip 员工使用手册')
  out.setAuthor('BeaconChip')
  out.setSubject('API Gateway Platform 员工使用手册')
  fs.writeFileSync(OUT, await out.save())
  console.log(`wrote ${OUT} (${out.getPageCount()} pages)`)
})().catch((e) => { console.error(e); process.exit(1) })
