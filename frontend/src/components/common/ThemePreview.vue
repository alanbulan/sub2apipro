<template>
  <span class="theme-preview" :class="['theme-preview-' + props.themeId]">
    <span class="preview-window">
      <span class="preview-topbar">
        <i class="preview-mark"></i>
        <i class="preview-nav"></i>
        <i class="preview-nav preview-nav-short"></i>
        <i class="preview-action"></i>
      </span>
      <span class="preview-stage">
        <span class="preview-sidebar" aria-hidden="true">
          <i></i><i></i><i></i><i></i>
        </span>
        <span class="preview-grid" aria-hidden="true">
          <i></i><i></i><i></i><i></i><i></i><i></i>
        </span>
        <i class="preview-shape preview-shape-a"></i>
        <i class="preview-shape preview-shape-b"></i>
        <i class="preview-shape preview-shape-c"></i>
        <span class="preview-copy">
          <i class="preview-copy-title"></i>
          <i class="preview-copy-line"></i>
          <i class="preview-copy-line preview-copy-line-short"></i>
          <span class="preview-copy-chip"><i></i></span>
          <i class="preview-copy-button"></i>
        </span>
        <span class="preview-panel">
          <span class="preview-panel-art">
            <i class="preview-art-shape"></i>
            <i class="preview-art-line"></i>
            <i class="preview-art-line preview-art-line-short"></i>
          </span>
          <i class="preview-panel-title"></i>
          <i class="preview-panel-line"></i>
          <i class="preview-panel-line preview-panel-line-short"></i>
          <i class="preview-panel-control"></i>
          <span class="preview-panel-dots"><i></i><i></i><i></i></span>
        </span>
        <span class="preview-detail">
          <i></i><i></i><i></i>
        </span>
        <span class="preview-terminal-lines" aria-hidden="true">
          <i></i><i></i><i></i><i></i><i></i>
        </span>
        <span class="preview-seal" aria-hidden="true"></span>
        <span class="preview-arch" aria-hidden="true"></span>
      </span>
    </span>
  </span>
</template>

<script setup lang="ts">
import type { AppThemeId } from '@/composables/useAppTheme'

const props = defineProps<{ themeId: AppThemeId }>()
</script>

<style scoped>
.theme-preview {
  --p-bg: #f7f7f8;
  --p-surface: #ffffff;
  --p-ink: #1f2937;
  --p-muted: #94a3b8;
  --p-primary: #64748b;
  --p-secondary: #cbd5e1;
  --p-border: rgb(15 23 42 / 14%);
  --p-shadow: rgb(15 23 42 / 12%);
  display: block;
  width: 100%;
  aspect-ratio: 1.78;
  overflow: hidden;
  border-radius: max(0.25rem, calc(var(--control-radius, 0.75rem) - 0.2rem));
  background: var(--p-bg);
}

.preview-window,
.preview-stage,
.preview-topbar,
.preview-copy,
.preview-panel,
.preview-detail {
  display: block;
}

.preview-window {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  color: var(--p-ink);
  background: var(--p-bg);
}

.preview-topbar {
  position: absolute;
  z-index: 3;
  inset: 0 0 auto;
  height: 18%;
  padding: 0 6%;
  border-bottom: 1px solid var(--p-border);
  background: var(--p-surface);
}

.preview-mark,
.preview-nav,
.preview-action,
.preview-copy-title,
.preview-copy-line,
.preview-copy-button,
.preview-panel-title,
.preview-panel-line,
.preview-panel-control,
.preview-detail i {
  position: absolute;
  display: block;
  border-radius: 999px;
  background: var(--p-secondary);
}

.preview-mark {
  top: 35%;
  left: 6%;
  width: 5%;
  aspect-ratio: 1;
  border-radius: 35%;
  background: var(--p-primary);
}

.preview-nav {
  top: 42%;
  left: 23%;
  width: 18%;
  height: 10%;
  opacity: 0.7;
}

.preview-nav-short {
  left: 45%;
  width: 11%;
}

.preview-action {
  top: 27%;
  right: 6%;
  width: 16%;
  height: 46%;
  border-radius: 0.25rem;
  background: var(--p-primary);
}

.preview-stage {
  position: absolute;
  inset: 18% 0 0;
}

.preview-copy {
  position: absolute;
  z-index: 2;
  top: 22%;
  left: 8%;
  width: 42%;
  height: 60%;
}

.preview-copy-title {
  position: relative;
  width: 76%;
  height: 13%;
  border-radius: 0.12rem;
  background: var(--p-ink);
}

.preview-copy-line {
  position: relative;
  width: 88%;
  height: 6%;
  margin-top: 9%;
}

.preview-copy-line-short {
  width: 62%;
  margin-top: 5%;
}

.preview-copy-button {
  position: relative;
  width: 32%;
  height: 16%;
  margin-top: 15%;
  border-radius: 0.2rem;
  background: var(--p-primary);
}

.preview-panel {
  position: absolute;
  z-index: 1;
  top: 17%;
  right: 8%;
  width: 37%;
  height: 64%;
  padding: 9% 7%;
  overflow: hidden;
  border: 1px solid var(--p-border);
  border-radius: 0.45rem;
  background: var(--p-surface);
  box-shadow: 0 0.35rem 0.8rem var(--p-shadow);
}

.preview-panel-title,
.preview-panel-line,
.preview-panel-control {
  position: relative;
  width: 68%;
  height: 8%;
  margin-bottom: 10%;
  border-radius: 0.1rem;
  background: var(--p-ink);
}

.preview-panel-line {
  width: 92%;
  height: 5%;
  margin-bottom: 7%;
  background: var(--p-muted);
}

.preview-panel-line-short {
  width: 72%;
}

.preview-panel-control {
  width: 48%;
  height: 17%;
  margin-top: 7%;
  border-radius: 0.2rem;
  background: var(--p-primary);
}

.preview-detail {
  position: absolute;
  z-index: 2;
  right: 9%;
  bottom: 9%;
  display: flex;
  gap: 4%;
  width: 32%;
}

.preview-detail i {
  position: relative;
  width: 29%;
  height: 0.22rem;
  background: var(--p-muted);
}

/* The following variants mirror the visual cues called out in each design
   reference: layout, material, type rhythm and interaction affordances. */
.theme-preview-minimalism {
  --p-bg: #f0f0f3;
  --p-surface: #ffffff;
  --p-ink: #0a0a0c;
  --p-muted: #6d6d76;
  --p-primary: #5e6ad2;
  --p-secondary: #d9d9de;
  --p-border: #e2e2e7;
  --p-shadow: rgb(10 10 12 / 12%);
}

.theme-preview-apple {
  --p-bg: #eef2f7;
  --p-surface: rgb(255 255 255 / 76%);
  --p-ink: #1d1d1f;
  --p-muted: #86868b;
  --p-primary: #007aff;
  --p-secondary: #64d2ff;
  --p-border: rgb(255 255 255 / 72%);
  --p-shadow: rgb(29 29 31 / 18%);
}

.theme-preview-apple .preview-window {
  background: radial-gradient(circle at 78% 24%, #c7edff 0 18%, transparent 42%), #eef2f7;
}

.theme-preview-apple .preview-panel {
  border-radius: 1rem;
  backdrop-filter: blur(6px);
}

.theme-preview-notion {
  --p-bg: #f7f7f5;
  --p-surface: #ffffff;
  --p-ink: #37352f;
  --p-muted: #a3a29e;
  --p-primary: #37352f;
  --p-secondary: #e3e2de;
  --p-border: #e3e2de;
  --p-shadow: rgb(55 53 47 / 8%);
}

.theme-preview-notion .preview-copy {
  left: 16%;
  width: 38%;
}

.theme-preview-notion .preview-panel {
  right: 15%;
  width: 36%;
  border-radius: 0.15rem;
  box-shadow: none;
}

.theme-preview-bento {
  --p-bg: #d7d7dc;
  --p-surface: #ffffff;
  --p-ink: #2d2d34;
  --p-muted: #858793;
  --p-primary: #2d2d34;
  --p-secondary: #e5e6eb;
  --p-border: transparent;
  --p-shadow: transparent;
}

.theme-preview-bento .preview-stage {
  display: grid;
  grid-template-columns: 1.35fr 0.8fr 0.9fr;
  grid-template-rows: 1fr 0.78fr;
  gap: 4%;
  padding: 7% 6%;
}

.theme-preview-bento .preview-copy,
.theme-preview-bento .preview-panel,
.theme-preview-bento .preview-detail {
  position: relative;
  inset: auto;
  width: auto;
  height: auto;
  min-width: 0;
  min-height: 0;
  padding: 12%;
  border: 0;
  border-radius: 0.45rem;
  background: var(--p-surface);
  box-shadow: none;
}

.theme-preview-bento .preview-copy {
  grid-row: span 2;
}

.theme-preview-bento .preview-panel {
  grid-column: 2 / span 2;
}

.theme-preview-bento .preview-detail {
  grid-column: 2 / span 2;
}

.theme-preview-bento .preview-copy-button,
.theme-preview-bento .preview-panel-control {
  margin-top: 20%;
}

.theme-preview-glass {
  --p-bg: #b9ddde;
  --p-surface: rgb(255 255 255 / 42%);
  --p-ink: #173b4b;
  --p-muted: #467080;
  --p-primary: #136f8a;
  --p-secondary: rgb(255 255 255 / 58%);
  --p-border: rgb(255 255 255 / 68%);
  --p-shadow: rgb(29 78 91 / 18%);
}

.theme-preview-glass .preview-window {
  background:
    radial-gradient(circle at 18% 70%, #f3b4d2 0 18%, transparent 38%),
    radial-gradient(circle at 82% 28%, #8ed9ed 0 22%, transparent 44%),
    #c6e5e5;
}

.theme-preview-glass .preview-panel {
  border-radius: 0.8rem;
  backdrop-filter: blur(5px);
}

.theme-preview-neoBrutalism {
  --p-bg: #ffdd00;
  --p-surface: #ff90e8;
  --p-ink: #000000;
  --p-muted: #000000;
  --p-primary: #90e8ff;
  --p-secondary: #ffffff;
  --p-border: #000000;
  --p-shadow: #000000;
}

.theme-preview-neoBrutalism .preview-window {
  background: #ffdd00;
}

.theme-preview-neoBrutalism .preview-panel {
  border-width: 2px;
  border-radius: 0;
  box-shadow: 0.3rem 0.3rem 0 #000;
}

.theme-preview-neoBrutalism .preview-copy-button {
  border-radius: 0;
  box-shadow: 0.2rem 0.2rem 0 #000;
}

.theme-preview-neoBrutalism .preview-shape-a {
  position: absolute;
  top: 7%;
  right: 10%;
  width: 12%;
  aspect-ratio: 1;
  background: #90e8ff;
  transform: rotate(12deg);
}

.theme-preview-swiss {
  --p-bg: #ffffff;
  --p-surface: #ffffff;
  --p-ink: #000000;
  --p-muted: #999999;
  --p-primary: #e30613;
  --p-secondary: #e3e3e3;
  --p-border: #d9d9d9;
  --p-shadow: transparent;
}

.theme-preview-swiss .preview-topbar {
  background: #000;
}

.theme-preview-swiss .preview-mark,
.theme-preview-swiss .preview-nav,
.theme-preview-swiss .preview-nav-short {
  background: #fff;
}

.theme-preview-swiss .preview-action {
  border-radius: 0;
  background: #e30613;
}

.theme-preview-swiss .preview-panel {
  border-radius: 0;
  box-shadow: none;
}

.theme-preview-editorial {
  --p-bg: #d9d2c4;
  --p-surface: #fffdf8;
  --p-ink: #1c1812;
  --p-muted: #8a7f6a;
  --p-primary: #3d372c;
  --p-secondary: #b8ae9d;
  --p-border: #d9d2c4;
  --p-shadow: rgb(28 24 18 / 10%);
}

.theme-preview-editorial .preview-window,
.theme-preview-editorial .preview-panel {
  font-family: Georgia, 'Times New Roman', serif;
}

.theme-preview-editorial .preview-topbar {
  background: #1c1812;
}

.theme-preview-editorial .preview-mark,
.theme-preview-editorial .preview-nav,
.theme-preview-editorial .preview-nav-short {
  background: #d9d2c4;
}

.theme-preview-editorial .preview-copy {
  width: 46%;
}

.theme-preview-editorial .preview-panel {
  border-radius: 0;
}

.theme-preview-skeuomorphism {
  --p-bg: #e9d9b7;
  --p-surface: #f7edcf;
  --p-ink: #4b2e1e;
  --p-muted: #9a7557;
  --p-primary: #74462c;
  --p-secondary: #d0af76;
  --p-border: #9e734e;
  --p-shadow: rgb(75 46 30 / 28%);
}

.theme-preview-skeuomorphism .preview-topbar {
  border-bottom: 2px dashed #d5b174;
  background: linear-gradient(#8e5d3b, #5b341f);
}

.theme-preview-skeuomorphism .preview-mark,
.theme-preview-skeuomorphism .preview-nav,
.theme-preview-skeuomorphism .preview-nav-short {
  background: #f5d69d;
}

.theme-preview-skeuomorphism .preview-window {
  background: repeating-linear-gradient(0deg, rgb(122 83 48 / 7%) 0 1px, transparent 1px 7px), var(--p-bg);
}

.theme-preview-skeuomorphism .preview-panel {
  border-color: #bd9b67;
  border-radius: 0.1rem;
  box-shadow: inset 0 0 0 2px rgb(255 255 255 / 34%), 0.25rem 0.35rem 0.5rem var(--p-shadow);
}

.theme-preview-flat {
  --p-bg: #f4f7f9;
  --p-surface: #ffffff;
  --p-ink: #203044;
  --p-muted: #718096;
  --p-primary: #2d89ef;
  --p-secondary: #60a917;
  --p-border: #d4dde6;
  --p-shadow: transparent;
}

.theme-preview-flat .preview-panel {
  border-radius: 0.15rem;
  box-shadow: none;
}

.theme-preview-flat .preview-copy-button {
  border-radius: 0.15rem;
}

.theme-preview-flat .preview-shape-a {
  position: absolute;
  top: 12%;
  left: 52%;
  width: 10%;
  height: 23%;
  background: var(--p-secondary);
}

.theme-preview-material {
  --p-bg: #fffbfe;
  --p-surface: #ffffff;
  --p-ink: #1d1b20;
  --p-muted: #625b71;
  --p-primary: #6750a4;
  --p-secondary: #e8def8;
  --p-border: #e7e0ec;
  --p-shadow: rgb(29 27 32 / 20%);
}

.theme-preview-material .preview-panel {
  border: 0;
  border-radius: 0.65rem;
  box-shadow: 0 0.35rem 0.7rem var(--p-shadow);
}

.theme-preview-material .preview-action {
  width: 13%;
  aspect-ratio: 1;
  height: auto;
  border-radius: 50%;
  box-shadow: 0 0.2rem 0.5rem var(--p-shadow);
}

.theme-preview-neumorphism {
  --p-bg: #e3e9f2;
  --p-surface: #e3e9f2;
  --p-ink: #263445;
  --p-muted: #607083;
  --p-primary: #6d7d92;
  --p-secondary: #b4bfd2;
  --p-border: transparent;
  --p-shadow: rgb(94 109 128 / 30%);
}

.theme-preview-neumorphism .preview-topbar {
  background: #e3e9f2;
  border-bottom: 0;
}

.theme-preview-neumorphism .preview-panel {
  border: 0;
  background: #e3e9f2;
  box-shadow: 0.28rem 0.28rem 0.55rem #b4bfd2, -0.28rem -0.28rem 0.55rem #ffffff;
}

.theme-preview-neumorphism .preview-copy-button {
  background: #e3e9f2;
  box-shadow: inset 0.18rem 0.18rem 0.35rem #b4bfd2, inset -0.18rem -0.18rem 0.35rem #ffffff;
}

.theme-preview-saas {
  --p-bg: #f5f7fb;
  --p-surface: #ffffff;
  --p-ink: #172033;
  --p-muted: #718096;
  --p-primary: #4f46e5;
  --p-secondary: #06b6d4;
  --p-border: #e2e8f0;
  --p-shadow: rgb(30 41 59 / 10%);
}

.theme-preview-saas .preview-copy-title {
  width: 92%;
}

.theme-preview-saas .preview-panel {
  top: 8%;
  height: 78%;
  border-radius: 0.3rem;
}

.theme-preview-saas .preview-panel-control {
  background: var(--p-secondary);
}

.theme-preview-enterprise {
  --p-bg: #f5f7fb;
  --p-surface: #ffffff;
  --p-ink: #172033;
  --p-muted: #5b6b82;
  --p-primary: #1d4ed8;
  --p-secondary: #0f766e;
  --p-border: #d7dee9;
  --p-shadow: rgb(23 32 51 / 9%);
}

.theme-preview-enterprise .preview-topbar {
  background: #1d4ed8;
}

.theme-preview-enterprise .preview-mark,
.theme-preview-enterprise .preview-nav,
.theme-preview-enterprise .preview-nav-short,
.theme-preview-enterprise .preview-action {
  background: #ffffff;
}

.theme-preview-enterprise .preview-panel {
  border-radius: 0.2rem;
}

.theme-preview-commerce {
  --p-bg: #fff8f1;
  --p-surface: #ffffff;
  --p-ink: #431407;
  --p-muted: #8b6b5a;
  --p-primary: #ea580c;
  --p-secondary: #db2777;
  --p-border: #fed7aa;
  --p-shadow: rgb(124 45 18 / 12%);
}

.theme-preview-commerce .preview-shape-a {
  position: absolute;
  top: 19%;
  right: 18%;
  width: 27%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: radial-gradient(circle at 35% 30%, #ffd9bf, #ea580c 68%);
}

.theme-preview-commerce .preview-panel {
  top: 13%;
  right: 10%;
  width: 40%;
  height: 72%;
}

.theme-preview-commerce .preview-panel-control {
  background: var(--p-secondary);
}

.theme-preview-darkTech {
  --p-bg: #050b13;
  --p-surface: #0b1624;
  --p-ink: #cffafe;
  --p-muted: #67e8f9;
  --p-primary: #06b6d4;
  --p-secondary: #a78bfa;
  --p-border: #164e63;
  --p-shadow: rgb(6 182 212 / 18%);
}

.theme-preview-darkTech .preview-window {
  background: radial-gradient(circle at 75% 15%, rgb(6 182 212 / 22%), transparent 42%), #050b13;
}

.theme-preview-darkTech .preview-topbar {
  background: #0b1624;
}

.theme-preview-darkTech .preview-panel {
  background: #0b1624;
  box-shadow: 0 0 0.8rem var(--p-shadow);
}

.theme-preview-playful {
  --p-bg: #fff7e8;
  --p-surface: #fffdf8;
  --p-ink: #4a1734;
  --p-muted: #a66b8a;
  --p-primary: #ec4899;
  --p-secondary: #f59e0b;
  --p-border: #f9a8d4;
  --p-shadow: rgb(245 158 11 / 24%);
}

.theme-preview-playful .preview-shape-a,
.theme-preview-playful .preview-shape-b,
.theme-preview-playful .preview-shape-c {
  position: absolute;
  border-radius: 50%;
}

.theme-preview-playful .preview-shape-a {
  top: 8%;
  right: 10%;
  width: 19%;
  aspect-ratio: 1;
  background: #60a5fa;
}

.theme-preview-playful .preview-shape-b {
  right: 31%;
  bottom: 8%;
  width: 10%;
  aspect-ratio: 1;
  background: #f472b6;
}

.theme-preview-playful .preview-panel {
  border-radius: 1rem;
  box-shadow: 0 0.45rem 0 var(--p-shadow);
}

.theme-preview-organic {
  --p-bg: #f6f3e8;
  --p-surface: #fbf8ee;
  --p-ink: #28351d;
  --p-muted: #637052;
  --p-primary: #637a2a;
  --p-secondary: #a16207;
  --p-border: #d1ddb8;
  --p-shadow: rgb(63 98 18 / 12%);
}

.theme-preview-organic .preview-window {
  background: radial-gradient(ellipse at 82% 32%, #d4e2b6 0 22%, transparent 43%), #f6f3e8;
}

.theme-preview-organic .preview-shape-a {
  position: absolute;
  bottom: -18%;
  left: 44%;
  width: 32%;
  height: 62%;
  border-radius: 62% 38% 55% 45%;
  background: #c4d5a1;
  transform: rotate(-18deg);
}

.theme-preview-organic .preview-panel {
  border-radius: 1rem 0.3rem 1.3rem 0.5rem;
}

.theme-preview-y2k {
  --p-bg: #dff4ff;
  --p-surface: rgb(255 255 255 / 78%);
  --p-ink: #24104f;
  --p-muted: #65558d;
  --p-primary: #ee64b4;
  --p-secondary: #65c9ff;
  --p-border: #c4b5fd;
  --p-shadow: rgb(109 40 217 / 18%);
}

.theme-preview-y2k .preview-window {
  background: linear-gradient(140deg, #dff4ff, #f7c9f1 52%, #c4b5fd);
}

.theme-preview-y2k .preview-copy-title {
  background: linear-gradient(180deg, #ffffff, #8795aa 45%, #ffffff 58%, #7c3aed);
  background-clip: text;
  -webkit-background-clip: text;
  color: transparent;
}

.theme-preview-y2k .preview-panel {
  border-radius: 1rem;
  backdrop-filter: blur(4px);
}

.theme-preview-y2k .preview-shape-a,
.theme-preview-y2k .preview-shape-b {
  position: absolute;
  border-radius: 999px;
  border: 2px solid rgb(255 255 255 / 68%);
  background: rgb(255 255 255 / 22%);
  box-shadow: inset 0.2rem 0.2rem 0.3rem rgb(255 255 255 / 70%);
}

.theme-preview-y2k .preview-shape-a {
  top: 5%;
  right: 9%;
  width: 18%;
  height: 22%;
}

.theme-preview-y2k .preview-shape-b {
  right: 28%;
  bottom: 7%;
  width: 10%;
  height: 13%;
}

.theme-preview-memphis {
  --p-bg: #fff5c8;
  --p-surface: #ffffff;
  --p-ink: #111827;
  --p-muted: #4b5563;
  --p-primary: #f43f5e;
  --p-secondary: #2563eb;
  --p-border: #111827;
  --p-shadow: #111827;
}

.theme-preview-memphis .preview-window {
  background-color: #fff5c8;
  background-image: radial-gradient(#f43f5e 1px, transparent 1px);
  background-size: 9px 9px;
}

.theme-preview-memphis .preview-panel {
  border-width: 2px;
  border-radius: 0.15rem;
  box-shadow: 0.25rem 0.25rem 0 #111827;
}

.theme-preview-memphis .preview-shape-a {
  position: absolute;
  top: 8%;
  right: 11%;
  width: 0;
  height: 0;
  border-right: 0.7rem solid transparent;
  border-bottom: 1.2rem solid #2563eb;
  border-left: 0.7rem solid transparent;
  transform: rotate(18deg);
}

.theme-preview-terminal {
  --p-bg: #020b07;
  --p-surface: #071a10;
  --p-ink: #c9d2c9;
  --p-muted: #6b7a6b;
  --p-primary: #4af626;
  --p-secondary: #86efac;
  --p-border: #166534;
  --p-shadow: rgb(74 246 38 / 16%);
}

.theme-preview-terminal .preview-window,
.theme-preview-terminal .preview-window * {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.theme-preview-terminal .preview-topbar {
  background: #071a10;
}

.theme-preview-terminal .preview-mark {
  border-radius: 0;
}

.theme-preview-terminal .preview-nav,
.theme-preview-terminal .preview-nav-short {
  background: #6b7a6b;
}

.theme-preview-terminal .preview-action {
  border-radius: 0;
  background: #4af626;
}

.theme-preview-terminal .preview-panel {
  border-radius: 0.15rem;
  background: #071a10;
  box-shadow: 0 0 0.75rem var(--p-shadow);
}

.theme-preview-wabiSabi {
  --p-bg: #f4eee3;
  --p-surface: #fbf8f1;
  --p-ink: #3f382f;
  --p-muted: #756c5f;
  --p-primary: #8f4631;
  --p-secondary: #d6cbb8;
  --p-border: rgb(63 56 47 / 18%);
  --p-shadow: rgb(63 56 47 / 10%);
}

.theme-preview-wabiSabi .preview-window {
  background: radial-gradient(rgb(117 108 95 / 10%) 0.7px, transparent 0.7px), #f4eee3;
  background-size: 5px 5px;
}

.theme-preview-wabiSabi .preview-topbar {
  background: #3f382f;
}

.theme-preview-wabiSabi .preview-mark,
.theme-preview-wabiSabi .preview-nav,
.theme-preview-wabiSabi .preview-nav-short {
  background: #d6cbb8;
}

.theme-preview-wabiSabi .preview-panel {
  top: 23%;
  right: 10%;
  width: 35%;
  transform: rotate(2deg);
}

.theme-preview-wabiSabi .preview-shape-a {
  position: absolute;
  top: 14%;
  right: 7%;
  width: 9%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: #b23a28;
}

.theme-preview-bauhaus {
  --p-bg: #fffdf5;
  --p-surface: #ffffff;
  --p-ink: #111827;
  --p-muted: #4b5563;
  --p-primary: #dc2626;
  --p-secondary: #1d4ed8;
  --p-border: #1f2937;
  --p-shadow: transparent;
}

.theme-preview-bauhaus .preview-window {
  background: #fffdf5;
}

.theme-preview-bauhaus .preview-topbar {
  background: #111827;
}

.theme-preview-bauhaus .preview-mark,
.theme-preview-bauhaus .preview-nav,
.theme-preview-bauhaus .preview-nav-short {
  background: #fffdf5;
}

.theme-preview-bauhaus .preview-panel {
  border-radius: 0;
  box-shadow: none;
}

.theme-preview-bauhaus .preview-shape-a {
  position: absolute;
  top: 6%;
  right: 9%;
  width: 18%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: #dc2626;
}

.theme-preview-bauhaus .preview-shape-b {
  position: absolute;
  right: 32%;
  bottom: 8%;
  width: 13%;
  height: 24%;
  background: #1d4ed8;
}

.theme-preview-bauhaus .preview-shape-c {
  position: absolute;
  top: 12%;
  left: 48%;
  width: 0;
  height: 0;
  border-right: 0.65rem solid transparent;
  border-bottom: 1.1rem solid #facc15;
  border-left: 0.65rem solid transparent;
}

.theme-preview-artDeco {
  --p-bg: #171717;
  --p-surface: #211f1d;
  --p-ink: #fef3c7;
  --p-muted: #d6b978;
  --p-primary: #b08d57;
  --p-secondary: #765a32;
  --p-border: #b08d57;
  --p-shadow: rgb(0 0 0 / 30%);
}

.theme-preview-artDeco .preview-window {
  background: #171717;
}

.theme-preview-artDeco .preview-topbar {
  border-bottom-color: #b08d57;
  background: #211f1d;
}

.theme-preview-artDeco .preview-mark,
.theme-preview-artDeco .preview-nav,
.theme-preview-artDeco .preview-nav-short {
  background: #b08d57;
}

.theme-preview-artDeco .preview-copy {
  left: 13%;
  width: 74%;
  text-align: center;
}

.theme-preview-artDeco .preview-copy-title,
.theme-preview-artDeco .preview-copy-line,
.theme-preview-artDeco .preview-copy-line-short,
.theme-preview-artDeco .preview-copy-button {
  left: 50%;
  transform: translateX(-50%);
}

.theme-preview-artDeco .preview-copy-title {
  width: 42%;
  background: #b08d57;
}

.theme-preview-artDeco .preview-copy-line {
  width: 62%;
  background: #765a32;
}

.theme-preview-artDeco .preview-copy-line-short {
  width: 38%;
}

.theme-preview-artDeco .preview-panel {
  top: 11%;
  right: 20%;
  width: 60%;
  height: 75%;
  border-color: #b08d57;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.theme-preview-artDeco .preview-panel::before,
.theme-preview-artDeco .preview-panel::after {
  position: absolute;
  content: '';
  border: 1px solid #b08d57;
}

.theme-preview-artDeco .preview-panel::before {
  inset: 8%;
}

.theme-preview-artDeco .preview-panel::after {
  inset: 17%;
}

.theme-preview-artDeco .preview-shape-a {
  position: absolute;
  top: 18%;
  left: 43%;
  width: 14%;
  height: 36%;
  border-top: 1px solid #b08d57;
  border-radius: 50% 50% 0 0;
  background: repeating-conic-gradient(from 270deg at 50% 100%, #b08d57 0 2deg, transparent 2deg 12deg);
}

/* Compact scenes follow the reference demos rather than reducing every theme
   to the same two-color swatch. The markup is intentionally text-free so the
   drawer remains legible in both Chinese and English locales. */
.preview-sidebar,
.preview-grid,
.preview-copy-chip,
.preview-panel-art,
.preview-panel-dots,
.preview-terminal-lines,
.preview-seal,
.preview-arch {
  display: none;
}

.preview-stage {
  isolation: isolate;
}

.preview-copy-chip {
  position: relative;
  width: 31%;
  height: 8%;
  margin-top: 10%;
  border: 1px solid var(--p-border);
  border-radius: 999px;
  background: var(--p-surface);
}

.preview-copy-chip i {
  display: block;
  width: 36%;
  height: 34%;
  margin: 33% 0 0 18%;
  border-radius: 999px;
  background: var(--p-primary);
}

.preview-panel-art {
  position: relative;
  overflow: hidden;
  height: 31%;
  margin: -3% -4% 11%;
  border-radius: 0.35rem;
  background: linear-gradient(135deg, color-mix(in srgb, var(--p-primary) 18%, var(--p-surface)), var(--p-secondary));
}

.preview-art-shape {
  position: absolute;
  top: 18%;
  right: 13%;
  width: 28%;
  height: 62%;
  border-radius: 50%;
  background: color-mix(in srgb, var(--p-primary) 64%, white);
}

.preview-art-line {
  position: absolute;
  left: 10%;
  bottom: 23%;
  width: 38%;
  height: 7%;
  border-radius: 999px;
  background: color-mix(in srgb, var(--p-ink) 55%, transparent);
}

.preview-art-line-short {
  bottom: 10%;
  width: 25%;
}

.preview-panel-dots {
  position: absolute;
  right: 7%;
  bottom: 8%;
  gap: 10%;
  width: 28%;
}

.preview-panel-dots i {
  display: inline-block;
  width: 18%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: var(--p-primary);
}

.preview-terminal-lines {
  position: absolute;
  z-index: 4;
  left: 8%;
  bottom: 10%;
  width: 43%;
}

.preview-terminal-lines i {
  display: block;
  width: 78%;
  height: 0.18rem;
  margin-top: 0.26rem;
  background: var(--p-muted);
}

.preview-terminal-lines i:nth-child(2) { width: 92%; }
.preview-terminal-lines i:nth-child(3) { width: 66%; background: var(--p-primary); }
.preview-terminal-lines i:nth-child(4) { width: 84%; }
.preview-terminal-lines i:nth-child(5) { width: 48%; background: var(--p-secondary); }

/* Minimalism: centered copy, quiet gray field, black action and one indigo
   signal, matching the reference's restrained product landing page. */
.theme-preview-minimalism .preview-window {
  background: #f8f8fa;
}

.theme-preview-minimalism .preview-topbar {
  background: rgb(255 255 255 / 88%);
}

.theme-preview-minimalism .preview-stage {
  background: linear-gradient(90deg, transparent 0 51%, rgb(94 106 210 / 5%) 51% 100%);
}

.theme-preview-minimalism .preview-copy {
  top: 20%;
  left: 8%;
  width: 47%;
}

.theme-preview-minimalism .preview-copy-chip {
  display: block;
}

.theme-preview-minimalism .preview-copy-button,
.theme-preview-minimalism .preview-panel-control {
  background: #0a0a0c;
}

.theme-preview-minimalism .preview-panel {
  top: 17%;
  height: 69%;
  border-radius: 0.85rem;
  box-shadow: 0 0.8rem 1.7rem rgb(31 32 40 / 12%);
}

/* Apple HIG: translucent system chrome, a photographic gradient device and a
   floating inspector instead of a generic dashboard card. */
.theme-preview-apple .preview-topbar {
  background: rgb(251 251 253 / 82%);
  backdrop-filter: blur(8px);
}

.theme-preview-apple .preview-stage {
  background: #e9e9ee;
}

.theme-preview-apple .preview-copy {
  top: 23%;
  left: 8%;
  width: 43%;
}

.theme-preview-apple .preview-copy-title {
  background: linear-gradient(90deg, #0090e3, #7d4fe0 55%, #e0396b);
}

.theme-preview-apple .preview-panel {
  top: 11%;
  right: 7%;
  width: 39%;
  height: 76%;
  padding: 8% 7%;
  background: rgb(250 250 252 / 70%);
  border-radius: 0.75rem;
  backdrop-filter: blur(14px) saturate(1.3);
  box-shadow: 0 0.75rem 1.7rem rgb(50 39 65 / 18%);
}

.theme-preview-apple .preview-panel-art {
  display: block;
  margin-bottom: 12%;
  background: linear-gradient(135deg, #0a1128, #1d2b64 46%, #7a5af8 80%, #e0396b);
}

.theme-preview-apple .preview-panel-art::after {
  position: absolute;
  top: 0;
  left: 42%;
  width: 20%;
  height: 8%;
  border-radius: 0 0 0.25rem 0.25rem;
  background: #2b2b2d;
  content: '';
}

/* Notion: a narrow workspace rail and a white writing canvas with almost no
   elevation. */
.theme-preview-notion .preview-topbar {
  display: none;
}

.theme-preview-notion .preview-stage {
  inset: 0;
  background: #f7f7f5;
}

.theme-preview-notion .preview-sidebar {
  position: absolute;
  z-index: 3;
  inset: 0 auto 0 0;
  display: grid;
  align-content: start;
  gap: 8%;
  width: 29%;
  padding: 12% 8%;
  border-right: 1px solid #ececea;
  background: #f7f7f5;
}

.theme-preview-notion .preview-sidebar::before {
  display: block;
  width: 52%;
  height: 9%;
  border-radius: 0.2rem;
  background: #37352f;
  content: '';
}

.theme-preview-notion .preview-sidebar i {
  display: block;
  width: 80%;
  height: 5%;
  border-radius: 0.12rem;
  background: #d8d7d2;
}

.theme-preview-notion .preview-sidebar i:first-child {
  background: #ebebe8;
}

.theme-preview-notion .preview-copy {
  top: 16%;
  left: 38%;
  width: 48%;
}

.theme-preview-notion .preview-copy-title {
  width: 64%;
  height: 16%;
  background: #37352f;
}

.theme-preview-notion .preview-copy-chip {
  display: block;
  width: 10%;
  height: 12%;
  margin-top: 13%;
  border: 0;
  border-radius: 0.2rem;
  background: #37352f;
}

.theme-preview-notion .preview-panel {
  top: 21%;
  right: 9%;
  width: 47%;
  height: 61%;
  padding: 9% 8%;
  border: 0;
  border-radius: 0.1rem;
  box-shadow: none;
}

/* Bento: visible unequal cells are the point of the composition. */
.theme-preview-bento .preview-topbar,
.theme-preview-bento .preview-copy,
.theme-preview-bento .preview-panel,
.theme-preview-bento .preview-detail {
  display: none;
}

.theme-preview-bento .preview-stage {
  inset: 0;
  display: block;
  padding: 8% 6%;
  background: #f5f5f7;
}

.theme-preview-bento .preview-grid {
  position: absolute;
  inset: 8% 6%;
  display: grid;
  grid-template-columns: 1.35fr 1fr 0.78fr;
  grid-template-rows: 1.12fr 0.78fr 0.58fr;
  gap: 4%;
}

.theme-preview-bento .preview-grid i {
  display: block;
  border-radius: 0.65rem;
  background: #ffffff;
  box-shadow: 0 0.12rem 0.3rem rgb(0 0 0 / 6%);
}

.theme-preview-bento .preview-grid i:first-child {
  grid-row: span 2;
  background: linear-gradient(140deg, #eeeaff, #f9f8ff 64%, #e5f6ff);
}

.theme-preview-bento .preview-grid i:nth-child(2) {
  grid-column: span 2;
  background: #15151a;
}

.theme-preview-bento .preview-grid i:nth-child(3) { background: #dff6ed; }
.theme-preview-bento .preview-grid i:nth-child(4) { background: #fff3d7; }
.theme-preview-bento .preview-grid i:nth-child(5) { background: #ffffff; }
.theme-preview-bento .preview-grid i:nth-child(6) { background: #15151a; }

/* Glassmorphism: saturated indigo-purple-pink-orange field and layered frosted
   panes, matching the reference rather than a pale blue card. */
.theme-preview-glass .preview-window {
  background: linear-gradient(115deg, #4338ca 0%, #7c3aed 40%, #db2777 75%, #f97316 110%);
}

.theme-preview-glass .preview-topbar {
  border-bottom-color: rgb(255 255 255 / 18%);
  background: rgb(20 18 58 / 38%);
  backdrop-filter: blur(12px);
}

.theme-preview-glass .preview-mark { background: #ffffff; }
.theme-preview-glass .preview-nav { background: rgb(255 255 255 / 60%); }

.theme-preview-glass .preview-stage {
  background:
    radial-gradient(circle at 12% 86%, rgb(255 183 214 / 58%) 0 13%, transparent 36%),
    radial-gradient(circle at 90% 22%, rgb(102 245 255 / 38%) 0 13%, transparent 39%);
}

.theme-preview-glass .preview-copy-title,
.theme-preview-glass .preview-copy-button {
  background: #ffffff;
}

.theme-preview-glass .preview-copy-line { background: rgb(255 255 255 / 60%); }

.theme-preview-glass .preview-panel {
  top: 12%;
  right: 8%;
  width: 40%;
  height: 72%;
  border-color: rgb(255 255 255 / 42%);
  border-radius: 0.8rem;
  background: rgb(255 255 255 / 16%);
  backdrop-filter: blur(16px);
  box-shadow: 0 0.8rem 1.7rem rgb(28 12 80 / 30%);
}

.theme-preview-glass .preview-panel-art {
  display: block;
  background: rgb(255 255 255 / 14%);
  border: 1px solid rgb(255 255 255 / 25%);
}

.theme-preview-glass .preview-panel-title,
.theme-preview-glass .preview-panel-line { background: rgb(255 255 255 / 72%); }

/* Neo-brutalism: dot field, white app shell, black outline and hard offset. */
.theme-preview-neoBrutalism .preview-window {
  background-color: #ffc62d;
  background-image: radial-gradient(#5f521d 0.7px, transparent 0.9px);
  background-size: 7px 7px;
}

.theme-preview-neoBrutalism .preview-topbar {
  left: 5%;
  right: 5%;
  top: 5%;
  height: 15%;
  border: 2px solid #050505;
  background: #fff8ee;
}

.theme-preview-neoBrutalism .preview-stage {
  inset: 20% 5% 5%;
  border: 2px solid #050505;
  background: #fff8ee;
  box-shadow: 0.35rem 0.35rem 0 #050505;
}

.theme-preview-neoBrutalism .preview-sidebar {
  position: absolute;
  z-index: 3;
  inset: 0 auto 0 0;
  display: grid;
  align-content: start;
  gap: 7%;
  width: 23%;
  padding: 9% 5%;
  border-right: 2px solid #050505;
  background: #eaff74;
}

.theme-preview-neoBrutalism .preview-sidebar i {
  display: block;
  width: 90%;
  height: 5%;
  border: 1px solid #050505;
  background: #fff8ee;
}

.theme-preview-neoBrutalism .preview-sidebar i:first-child {
  height: 10%;
  box-shadow: 0.12rem 0.12rem 0 #050505;
}

.theme-preview-neoBrutalism .preview-copy {
  top: 13%;
  left: 29%;
  width: 39%;
}

.theme-preview-neoBrutalism .preview-copy-title,
.theme-preview-neoBrutalism .preview-copy-line,
.theme-preview-neoBrutalism .preview-copy-line-short,
.theme-preview-neoBrutalism .preview-panel-title,
.theme-preview-neoBrutalism .preview-panel-line {
  border-radius: 0;
  background: #050505;
}

.theme-preview-neoBrutalism .preview-copy-button,
.theme-preview-neoBrutalism .preview-panel-control {
  border: 2px solid #050505;
  border-radius: 0;
  background: #ff8ed8;
  box-shadow: 0.2rem 0.2rem 0 #050505;
}

.theme-preview-neoBrutalism .preview-panel {
  top: 13%;
  right: 7%;
  width: 30%;
  height: 69%;
  border: 2px solid #050505;
  border-radius: 0;
  background: #87dcf2;
  box-shadow: 0.2rem 0.2rem 0 #050505;
}

.theme-preview-neoBrutalism .preview-panel-art {
  display: block;
  margin-bottom: 10%;
  border: 2px solid #050505;
  border-radius: 0;
  background: #ff8ed8;
}

/* Swiss: strict vertical grid, black header, red signal square. */
.theme-preview-swiss .preview-window {
  background: #ffffff;
}

.theme-preview-swiss .preview-topbar {
  background: #000000;
  border-bottom: 0;
}

.theme-preview-swiss .preview-mark,
.theme-preview-swiss .preview-nav,
.theme-preview-swiss .preview-nav-short { background: #ffffff; }

.theme-preview-swiss .preview-stage {
  background:
    linear-gradient(90deg, transparent 24.5%, #f0f0f0 24.5% 25%, transparent 25% 49.5%, #f0f0f0 49.5% 50%, transparent 50% 74.5%, #f0f0f0 74.5% 75%, transparent 75%),
    #ffffff;
}

.theme-preview-swiss .preview-copy {
  top: 18%;
  left: 8%;
  width: 56%;
}

.theme-preview-swiss .preview-copy-title {
  width: 62%;
  height: 21%;
  border-radius: 0;
  background: #000000;
}

.theme-preview-swiss .preview-copy-line,
.theme-preview-swiss .preview-copy-line-short { border-radius: 0; background: #555555; }

.theme-preview-swiss .preview-copy-button {
  width: 13%;
  aspect-ratio: 1;
  height: auto;
  border-radius: 0;
  background: #e30613;
}

.theme-preview-swiss .preview-panel {
  top: 15%;
  right: 9%;
  width: 27%;
  height: 68%;
  border: 0;
  border-left: 0.35rem solid #e30613;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

/* Editorial: warm paper, centered masthead lines and a split column rhythm. */
.theme-preview-editorial .preview-window {
  background: #faf7f1;
}

.theme-preview-editorial .preview-topbar {
  background: #faf7f1;
  border-bottom: 3px double #b8ac93;
}

.theme-preview-editorial .preview-mark,
.theme-preview-editorial .preview-nav,
.theme-preview-editorial .preview-nav-short { background: #8a7f6a; }

.theme-preview-editorial .preview-stage::before {
  position: absolute;
  top: 7%;
  left: 35%;
  width: 1px;
  height: 78%;
  background: #e8e2d4;
  content: '';
}

.theme-preview-editorial .preview-copy {
  top: 14%;
  left: 10%;
  width: 73%;
}

.theme-preview-editorial .preview-copy-title {
  left: 15%;
  width: 70%;
  height: 18%;
  border-radius: 0;
  background: #1c1812;
}

.theme-preview-editorial .preview-copy-line,
.theme-preview-editorial .preview-copy-line-short {
  border-radius: 0;
  background: #8a7f6a;
}

.theme-preview-editorial .preview-panel {
  top: 44%;
  right: 12%;
  width: 27%;
  height: 34%;
  border: 0;
  border-top: 3px double #b8ac93;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

/* Skeuomorphism: linen grain, leather strip and yellow ruled paper. */
.theme-preview-skeuomorphism .preview-window {
  background:
    repeating-linear-gradient(0deg, rgb(122 83 48 / 7%) 0 1px, transparent 1px 5px),
    repeating-linear-gradient(90deg, rgb(255 255 255 / 20%) 0 1px, transparent 1px 7px),
    #e9d9b7;
}

.theme-preview-skeuomorphism .preview-topbar {
  height: 20%;
  border-bottom: 2px dashed #d5b174;
  background: linear-gradient(#8e5d3b, #5b341f);
}

.theme-preview-skeuomorphism .preview-mark,
.theme-preview-skeuomorphism .preview-nav,
.theme-preview-skeuomorphism .preview-nav-short { background: #f5d69d; }

.theme-preview-skeuomorphism .preview-stage {
  inset: 20% 0 0;
}

.theme-preview-skeuomorphism .preview-copy {
  top: 13%;
  left: 8%;
  width: 47%;
  padding: 5% 6%;
  background: repeating-linear-gradient(#f7edcf 0 12%, #d8c99e 12.5% 13%, #f7edcf 13.5% 25%);
  box-shadow: 0.25rem 0.35rem 0.5rem rgb(75 46 30 / 28%);
}

.theme-preview-skeuomorphism .preview-copy-title {
  width: 55%;
  background: #4b2e1e;
}

.theme-preview-skeuomorphism .preview-panel {
  top: 17%;
  right: 8%;
  width: 35%;
  height: 63%;
  border-color: #bd9b67;
  border-radius: 0.1rem;
  background: #f7edcf;
  box-shadow: inset 0 0 0 2px rgb(255 255 255 / 34%), 0.25rem 0.35rem 0.5rem rgb(75 46 30 / 28%);
}

/* Flat: Metro-like tiles, no gradients and no elevation. */
.theme-preview-flat .preview-window { background: #f3f5f7; }
.theme-preview-flat .preview-topbar { background: #1ba1e2; border-color: #1685be; }
.theme-preview-flat .preview-mark,
.theme-preview-flat .preview-nav,
.theme-preview-flat .preview-nav-short { background: #ffffff; }

.theme-preview-flat .preview-stage {
  inset: 18% 0 0;
  padding: 7%;
  background: #f3f5f7;
}

.theme-preview-flat .preview-grid {
  position: absolute;
  inset: 10% 7%;
  display: grid;
  grid-template-columns: 1.3fr 1fr 0.8fr;
  grid-template-rows: 1fr 0.72fr;
  gap: 4%;
}

.theme-preview-flat .preview-grid i {
  display: block;
  border-radius: 0;
  background: #1ba1e2;
}

.theme-preview-flat .preview-grid i:nth-child(2) { background: #60a917; }
.theme-preview-flat .preview-grid i:nth-child(3) { background: #fa6800; }
.theme-preview-flat .preview-grid i:nth-child(4) { background: #e51400; }
.theme-preview-flat .preview-grid i:nth-child(5) { background: #68217a; }
.theme-preview-flat .preview-grid i:nth-child(6) { background: #00aba9; }

.theme-preview-flat .preview-copy,
.theme-preview-flat .preview-panel,
.theme-preview-flat .preview-detail { display: none; }

/* Material: blue app bar, elevated cards, and a pink floating action button. */
.theme-preview-material .preview-topbar {
  background: #1976d2;
  border-bottom: 0;
}

.theme-preview-material .preview-mark,
.theme-preview-material .preview-nav,
.theme-preview-material .preview-nav-short { background: #ffffff; }

.theme-preview-material .preview-stage { background: #f5f5f5; }

.theme-preview-material .preview-copy {
  top: 18%;
  left: 8%;
  width: 47%;
}

.theme-preview-material .preview-panel {
  top: 16%;
  right: 8%;
  width: 36%;
  height: 65%;
  border: 0;
  border-radius: 0.55rem;
  background: #ffffff;
  box-shadow: 0 0.35rem 0.7rem rgb(0 0 0 / 18%);
}

.theme-preview-material .preview-panel-art {
  display: block;
  background: #e3f2fd;
}

.theme-preview-material .preview-action {
  z-index: 5;
  top: auto;
  right: 10%;
  bottom: 9%;
  width: 15%;
  height: auto;
  aspect-ratio: 1;
  border-radius: 50%;
  background: #e91e63;
  box-shadow: 0 0.25rem 0.5rem rgb(0 0 0 / 26%);
}

/* Neumorphism: the surface and the controls share one blue-gray material. */
.theme-preview-neumorphism .preview-window,
.theme-preview-neumorphism .preview-topbar,
.theme-preview-neumorphism .preview-stage { background: #e3e9f2; }

.theme-preview-neumorphism .preview-topbar { border-bottom: 0; }

.theme-preview-neumorphism .preview-panel {
  border: 0;
  border-radius: 0.8rem;
  background: #e3e9f2;
  box-shadow: 0.28rem 0.28rem 0.55rem #b4bfd2, -0.28rem -0.28rem 0.55rem #ffffff;
}

.theme-preview-neumorphism .preview-copy-button {
  background: #e3e9f2;
  box-shadow: inset 0.18rem 0.18rem 0.35rem #b4bfd2, inset -0.18rem -0.18rem 0.35rem #ffffff;
}

/* SaaS and enterprise separate marketing hierarchy from product operations. */
.theme-preview-saas .preview-window { background: #f8f9fd; }
.theme-preview-saas .preview-topbar { background: #ffffff; }
.theme-preview-saas .preview-copy { width: 45%; }
.theme-preview-saas .preview-copy-title { width: 92%; }
.theme-preview-saas .preview-panel {
  top: 8%;
  height: 78%;
  border-radius: 0.35rem;
  background: #25213b;
  box-shadow: 0 0.45rem 1rem rgb(41 34 68 / 22%);
}

.theme-preview-saas .preview-panel-art {
  display: block;
  background: radial-gradient(circle at 73% 30%, #9a7cff 0 12%, transparent 13%), #332c53;
}

.theme-preview-saas .preview-panel-title,
.theme-preview-saas .preview-panel-line { background: rgb(255 255 255 / 78%); }
.theme-preview-saas .preview-panel-control { background: #9a7cff; }

.theme-preview-enterprise .preview-window { background: #f4f7f5; }
.theme-preview-enterprise .preview-topbar { background: #173f3b; }
.theme-preview-enterprise .preview-mark,
.theme-preview-enterprise .preview-nav,
.theme-preview-enterprise .preview-nav-short,
.theme-preview-enterprise .preview-action { background: #ffffff; }
.theme-preview-enterprise .preview-stage { background: #d9e5e0; }
.theme-preview-enterprise .preview-copy { top: 18%; left: 8%; width: 44%; }
.theme-preview-enterprise .preview-panel { border-radius: 0.2rem; background: rgb(255 255 255 / 88%); }
.theme-preview-enterprise .preview-panel-art { display: block; background: #c4d2cd; }

/* Commerce uses a warm product studio, not a generic orange dashboard. */
.theme-preview-commerce .preview-window { background: #f4efe7; }
.theme-preview-commerce .preview-topbar { background: #fffaf3; }
.theme-preview-commerce .preview-stage { background: linear-gradient(135deg, #d3c1ad, #eee1cf); }
.theme-preview-commerce .preview-copy { top: 17%; left: 8%; width: 43%; }
.theme-preview-commerce .preview-copy-title { background: #2b3a31; }
.theme-preview-commerce .preview-panel {
  top: 10%;
  right: 9%;
  width: 37%;
  height: 75%;
  border-radius: 0.3rem;
  background: rgb(255 250 243 / 84%);
}
.theme-preview-commerce .preview-panel-art { display: block; background: linear-gradient(150deg, #dcc7ad, #9e8063); }
.theme-preview-commerce .preview-art-shape { background: #2b3a31; }

/* Dark UI and terminal keep their high contrast instead of inheriting light
   utility colors from the application shell. */
.theme-preview-darkTech .preview-window { background: #0b0f15; }
.theme-preview-darkTech .preview-topbar { background: #111721; }
.theme-preview-darkTech .preview-stage { background: radial-gradient(circle at 78% 15%, rgb(38 207 160 / 22%), transparent 42%), #0b0f15; }
.theme-preview-darkTech .preview-panel { background: #111721; box-shadow: 0 0 0.8rem rgb(38 207 160 / 20%); }
.theme-preview-darkTech .preview-panel-art { display: block; background: #151e29; }
.theme-preview-darkTech .preview-panel-title { background: #d18cff; }
.theme-preview-darkTech .preview-panel-line { background: #7f8a99; }

.theme-preview-terminal .preview-window,
.theme-preview-terminal .preview-stage,
.theme-preview-terminal .preview-topbar { background: #0b0f0b; }
.theme-preview-terminal .preview-topbar { border-bottom-color: #273227; }
.theme-preview-terminal .preview-mark { border-radius: 0; background: #4af626; }
.theme-preview-terminal .preview-nav,
.theme-preview-terminal .preview-nav-short { background: #8ca08c; }
.theme-preview-terminal .preview-copy { top: 16%; left: 8%; width: 45%; }
.theme-preview-terminal .preview-copy-title,
.theme-preview-terminal .preview-copy-line { border-radius: 0; background: #c9d2c9; }
.theme-preview-terminal .preview-copy-button { border-radius: 0; background: #4af626; }
.theme-preview-terminal .preview-panel { background: #101410; border-color: #273227; box-shadow: 0 0 0.75rem rgb(74 246 38 / 16%); }
.theme-preview-terminal .preview-panel-art { display: block; border: 1px solid #273227; background: linear-gradient(#101410, #0b0f0b); }
.theme-preview-terminal .preview-terminal-lines { display: block; }

/* Playful, organic and Y2K are shape-led, so the decorative layer carries the
   identity while the common content remains quiet. */
.theme-preview-playful .preview-window { background: #fff6c8; }
.theme-preview-playful .preview-topbar { background: #fff6c8; border-bottom: 2px solid #2b2330; }
.theme-preview-playful .preview-stage { background: #fff6c8; }
.theme-preview-playful .preview-shape-a { background: #8d70f5; transform: rotate(-12deg); box-shadow: 0.2rem 0.2rem 0 #2b2330; }
.theme-preview-playful .preview-shape-b { background: #a8ef73; box-shadow: 0.15rem 0.15rem 0 #2b2330; }
.theme-preview-playful .preview-shape-c { display: block; position: absolute; right: 18%; bottom: 13%; width: 13%; height: 19%; border: 2px solid #2b2330; background: #ff7254; transform: rotate(8deg); }
.theme-preview-playful .preview-copy-title { background: #2b2330; }
.theme-preview-playful .preview-panel { border: 2px solid #2b2330; border-radius: 0.9rem; background: #ffdf58; box-shadow: 0.28rem 0.28rem 0 #2b2330; transform: rotate(2deg); }

.theme-preview-organic .preview-window { background: #eee6d9; }
.theme-preview-organic .preview-topbar { background: #eee6d9; }
.theme-preview-organic .preview-stage { background: linear-gradient(145deg, #9eae8d, #526c56); }
.theme-preview-organic .preview-shape-a { display: block; position: absolute; left: 43%; bottom: -20%; width: 34%; height: 70%; border-radius: 62% 38% 55% 45%; background: #d5c49c; transform: rotate(-18deg); }
.theme-preview-organic .preview-copy-title { background: #29402f; }
.theme-preview-organic .preview-panel { border-radius: 1rem 0.3rem 1.3rem 0.5rem; background: rgb(239 232 217 / 84%); }
.theme-preview-organic .preview-panel-art { display: block; background: #52735a; }

.theme-preview-y2k .preview-window { background: linear-gradient(140deg, #dff4ff, #f7c9f1 52%, #c4b5fd); }
.theme-preview-y2k .preview-topbar { background: rgb(255 255 255 / 60%); backdrop-filter: blur(10px); }
.theme-preview-y2k .preview-stage { background: radial-gradient(circle at 17% 82%, rgb(255 255 255 / 64%) 0 9%, transparent 10%), transparent; }
.theme-preview-y2k .preview-shape-a,
.theme-preview-y2k .preview-shape-b { display: block; position: absolute; border: 2px solid rgb(255 255 255 / 80%); border-radius: 999px; background: rgb(255 255 255 / 24%); box-shadow: inset 0.18rem 0.18rem 0.35rem rgb(255 255 255 / 70%); }
.theme-preview-y2k .preview-shape-a { top: 5%; right: 10%; width: 20%; height: 24%; }
.theme-preview-y2k .preview-shape-b { right: 29%; bottom: 7%; width: 11%; height: 15%; }
.theme-preview-y2k .preview-copy-title { background: linear-gradient(180deg, #ffffff, #8795aa 45%, #ffffff 58%, #7c3aed); }
.theme-preview-y2k .preview-panel { border-radius: 1rem; background: rgb(255 255 255 / 64%); backdrop-filter: blur(8px); box-shadow: inset 0 0.12rem 0 rgb(255 255 255 / 80%), 0 0.55rem 1rem rgb(109 40 217 / 18%); }

/* Memphis: cream dot field with independent triangle, circle and ring forms. */
.theme-preview-memphis .preview-window { background-color: #fff5c8; background-image: radial-gradient(#f43f5e 0.7px, transparent 0.8px); background-size: 6px 6px; }
.theme-preview-memphis .preview-topbar { background: #fff5c8; border-bottom: 2px solid #111827; }
.theme-preview-memphis .preview-stage { background: transparent; }
.theme-preview-memphis .preview-shape-a { display: block; position: absolute; top: 8%; right: 11%; width: 0; height: 0; border-right: 0.7rem solid transparent; border-bottom: 1.2rem solid #2563eb; border-left: 0.7rem solid transparent; transform: rotate(18deg); }
.theme-preview-memphis .preview-shape-b { display: block; position: absolute; right: 28%; bottom: 7%; width: 13%; height: 20%; border: 2px solid #00a6a6; border-radius: 50%; background: transparent; }
.theme-preview-memphis .preview-shape-c { display: block; position: absolute; left: 45%; top: 10%; width: 11%; height: 14%; border-radius: 50%; background: #ffd23f; }
.theme-preview-memphis .preview-panel { border: 2px solid #111827; border-radius: 0.15rem; background: #ffffff; box-shadow: 0.25rem 0.25rem 0 #111827; }
.theme-preview-memphis .preview-panel-control { background: #3d5af1; }

/* Wabi-sabi: quiet paper grain and a single vermilion seal. */
.theme-preview-wabiSabi .preview-window { background: radial-gradient(rgb(117 108 95 / 10%) 0.6px, transparent 0.7px), #f4eee3; background-size: 5px 5px; }
.theme-preview-wabiSabi .preview-topbar { background: #3f382f; }
.theme-preview-wabiSabi .preview-mark,
.theme-preview-wabiSabi .preview-nav,
.theme-preview-wabiSabi .preview-nav-short { background: #d6cbb8; }
.theme-preview-wabiSabi .preview-copy-title { background: #3f382f; }
.theme-preview-wabiSabi .preview-panel { top: 19%; right: 10%; width: 36%; height: 64%; border-radius: 0; background: #fbf8f1; transform: rotate(2deg); box-shadow: 0.25rem 0.4rem 0.8rem rgb(63 56 47 / 14%); }
.theme-preview-wabiSabi .preview-panel::before { position: absolute; inset: 11% 0 auto; border-top: 1px solid rgb(178 58 40 / 72%); content: ''; }
.theme-preview-wabiSabi .preview-seal { position: absolute; z-index: 4; top: 25%; right: 17%; display: block; width: 12%; aspect-ratio: 1; border-radius: 50%; background: #b23a28; box-shadow: inset 0 0 0 2px rgb(244 238 227 / 26%); }

/* Bauhaus: each primary geometric form has its own place in the layout. */
.theme-preview-bauhaus .preview-window { background: #f4efe6; }
.theme-preview-bauhaus .preview-topbar { background: #f4efe6; border-bottom: 2px solid #1d2129; }
.theme-preview-bauhaus .preview-mark { border-radius: 50%; background: #e64433; }
.theme-preview-bauhaus .preview-shape-a { display: block; position: absolute; top: 12%; left: 43%; width: 19%; aspect-ratio: 1; border-radius: 50%; background: #e64433; }
.theme-preview-bauhaus .preview-shape-b { display: block; position: absolute; right: 10%; top: 11%; width: 14%; height: 24%; background: #2b5fc7; }
.theme-preview-bauhaus .preview-shape-c { display: block; position: absolute; right: 29%; bottom: 10%; width: 0; height: 0; border-right: 0.7rem solid transparent; border-bottom: 1.2rem solid #f0b41e; border-left: 0.7rem solid transparent; }
.theme-preview-bauhaus .preview-copy { top: 42%; left: 10%; width: 74%; }
.theme-preview-bauhaus .preview-copy-title { background: #1d2129; }
.theme-preview-bauhaus .preview-panel { top: 40%; right: 8%; width: 28%; height: 33%; border-radius: 0; border: 2px solid #1d2129; background: transparent; box-shadow: none; }

/* Art Deco: stepped arches and a centered sunburst create symmetry. */
.theme-preview-artDeco .preview-window { background: #14161c; }
.theme-preview-artDeco .preview-topbar { background: #211f1d; border-bottom-color: rgb(201 162 39 / 60%); }
.theme-preview-artDeco .preview-mark,
.theme-preview-artDeco .preview-nav,
.theme-preview-artDeco .preview-nav-short { background: #c9a227; }
.theme-preview-artDeco .preview-copy { top: 48%; left: 15%; width: 70%; text-align: center; }
.theme-preview-artDeco .preview-copy-title,
.theme-preview-artDeco .preview-copy-line,
.theme-preview-artDeco .preview-copy-line-short,
.theme-preview-artDeco .preview-copy-button { left: 50%; transform: translateX(-50%); background: #c9a227; }
.theme-preview-artDeco .preview-copy-title { width: 44%; }
.theme-preview-artDeco .preview-copy-line { width: 61%; background: #765a32; }
.theme-preview-artDeco .preview-panel { top: 10%; right: 20%; width: 60%; height: 74%; border-color: #c9a227; border-radius: 0; background: transparent; box-shadow: none; }
.theme-preview-artDeco .preview-panel::before,
.theme-preview-artDeco .preview-panel::after { position: absolute; left: 50%; transform: translateX(-50%); border: 1px solid rgb(201 162 39 / 72%); border-bottom: 0; border-radius: 999px 999px 0 0; content: ''; }
.theme-preview-artDeco .preview-panel::before { bottom: 0; width: 76%; height: 64%; }
.theme-preview-artDeco .preview-panel::after { bottom: 0; width: 52%; height: 49%; }
.theme-preview-artDeco .preview-arch { position: absolute; left: 43%; bottom: 18%; display: block; width: 14%; height: 28%; border-radius: 50% 50% 0 0; background: repeating-conic-gradient(from -90deg at 50% 100%, rgb(201 162 39 / 62%) 0 7deg, transparent 7deg 18deg); }

@media (prefers-reduced-motion: reduce) {
  .theme-preview * {
    transition: none !important;
  }
}
</style>
