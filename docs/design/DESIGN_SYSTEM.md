# Prism Design System v2.0

## 1. Overview & Creative North Star

**Creative North Star: The Financial Instrument**

Prism is a Soroban-first block explorer that feels like a high-end financial tool — not a dashboard, not a marketing site. We draw from Ramp's receipt-level clarity, Etherscan's data scannability, and Token Terminal's metric focus to build an interface where complex blockchain data becomes immediately legible.

The guiding principle is **functional density**: every pixel must answer a question. We achieve information hierarchy through typography scale and weight — not through card grids, colored containers, or decorative elements. The design has exactly two visual zones per page: the **header** (identity + hero number + context) and the **content** (tables, key-value rows, data).

### Stack
- **Go 1.25** + **templ** templates + **htmx** for server-rendered partials
- **Tailwind CSS** with custom `text-2xs` token (0.625rem)
- **Instrument Sans** (UI) + **JetBrains Mono** (data)

---

## 2. Colors & Semantic Logic

Color conveys meaning, never decoration. Every color token has a defined semantic role.

### Neutrals (Semantic Tokens)

Neutral colors are defined as CSS custom properties and mapped to Tailwind utilities via `@theme`. They automatically adapt in dark mode.

| Token | Tailwind Class | Light | Dark | Role |
|-------|---------------|-------|------|------|
| Page background | `bg-surface-page` | `#FAFAFA` | `#09090B` | Page-level background |
| Card background | `bg-surface-card` | `#FFFFFF` | `#141414` | Cards, panels, nav |
| Subtle background | `bg-surface-subtle` | `#F3F4F6` | `#1E1E1E` | Table headers, hover states |
| Default border | `border-border-default` | `#E5E7EB` | `#2A2A2A` | Card borders, dividers |
| Subtle border | `border-border-subtle` | `#F3F4F6` | `#1E1E1E` | Inner dividers |
| Primary text | `text-text-primary` | `#111111` | `#F3F4F6` | Headings, primary content |
| Strong text | `text-text-strong` | `#374151` | `#D1D5DB` | Secondary headings, labels |
| Body text | `text-text-body` | `#6B7280` | `#9CA3AF` | Body copy, descriptions |
| Muted text | `text-text-muted` | `#9CA3AF` | `#6B7280` | Labels, placeholders, separators |

**Migration**: Do not use hardcoded `bg-white`, `bg-gray-50`, `border-gray-200`, `text-gray-900`, etc. Always use the semantic tokens above.

### Semantic Colors
| Color | Token | Role |
|-------|-------|------|
| **Emerald** | `#10B981` (500) | Primary / Success / Live. Network pulse, linked addresses, positive changes, healthy states. All clickable blockchain data uses emerald-700 — never blue. |
| **Violet** | `#7C3AED` (600) | Soroban / Smart Contracts. Contract calls, SAC assets, Soroban events, function names. This creates instant distinction between classic and programmable activity. |
| **Amber** | `#F59E0B` (500) | Warning / Caution. Metrics that are concerning but not critical (e.g., 57.3% crawler rejection, slow ledger close). |
| **Red** | `#EF4444` (500) | Error / Failed. Failed transactions, critical TTL, evicted storage entries. |
| **Cyan** | `#06B6D4` | Transfer events, lending badges. |

### Color Rules
- **Emerald links** — All clickable blockchain data (addresses, hashes, ledger numbers) use `text-emerald-700`. Never blue.
- **Violet = Soroban** — Contract calls, Soroban events, and smart contract badges always use violet. Color is on the *values*, not the *containers* — no violet-tinted card backgrounds.
- **No decorative color** — Color always conveys meaning (status, type, interaction).
- **Amber threshold** — Amber for concerning-but-not-critical. Red only for failures and errors.

### Dark Mode

The explorer supports a three-way theme toggle: **System → Light → Dark**, persisted in `localStorage('prism_theme')`.

**Toggle behavior**:
- **System**: Follows OS `prefers-color-scheme` preference
- **Light**: Forces light mode regardless of OS
- **Dark**: Forces dark mode regardless of OS
- Toggle button appears in the top nav between search and network selector
- Icons: monitor (system), sun (light), moon (dark)

**Implementation**: A synchronous `<script>` in `<head>` reads `localStorage` and applies `.dark` class to `<html>` before first paint, preventing FOUC.

**Semantic colors in dark mode**: Emerald, violet, amber, red, cyan stay at their standard Tailwind values. Badge tinted backgrounds use `dark:bg-{color}-950/30` and `dark:ring-{color}-800` variants.

**Inverted elements**: Buttons and pagination that use `bg-gray-900 text-white` in light mode flip to `dark:bg-gray-100 dark:text-gray-900`.

**Code blocks**: `bg-[#1E1E2E]` backgrounds are already dark — no change needed.

---

## 3. Typography

We pair a geometric-humanist sans-serif with a technical monospace to balance humanity with precision.

### Instrument Sans — UI & Body
| Style | Size | Weight | Use |
|-------|------|--------|-----|
| Page title | 24px | 700 | `text-2xl font-bold tracking-tight` |
| Section title | 18px | 700 | `text-lg font-bold` |
| Hero number | 36px | 700 | `text-4xl font-bold tabular` |
| Body | 14px | 400 | `text-sm` |
| Table cell | 12px | 400 | `text-xs` |
| Category label | 10px | 600 | `text-2xs font-semibold uppercase tracking-wider text-gray-400` |

### JetBrains Mono — Data & Code
| Style | Size | Use |
|-------|------|-----|
| Ledger numbers | 14px | `font-mono text-sm tabular` |
| Linked addresses | 12px | `font-mono text-xs text-emerald-700` |
| Hash fragments | 11px | `font-mono text-2xs text-gray-400` |
| Hostnames/technical | 10px | `font-mono text-2xs text-gray-500` |

### Type Rules
- **Mono for all blockchain data** — Addresses, hashes, ledger numbers, amounts, fees, sequences, public keys.
- **Sans for all UI text** — Labels, descriptions, navigation, buttons, headings.
- **`tabular-nums` required** — All numeric data uses `font-variant-numeric: tabular-nums` via the `.tabular` class so columns align.
- **Address truncation** — Format: first 5-6 chars + "..." + last 4-5 chars. Always pair with a copy-to-clipboard button.
- **Number formatting** — Comma separators for integers >= 1,000. Unit suffixes (K, M) only in inline stats and headers.

---

## 4. Spacing & Layout

### Page Structure
| Property | Value |
|----------|-------|
| Top nav height | 56px (`h-14`), sticky |
| Content max-width | 1320px |
| Content padding | 32px (`px-8`), 16px mobile (`px-4`) |
| Slide-out panel width | 480px |
| Page background | `#FAFAFA` |

### Component Spacing
| Property | Value |
|----------|-------|
| Card padding | 16-20px (`p-4` to `p-5`) |
| Card radius | 12px (`rounded-xl`) |
| Card border | 1px `border-gray-200` |
| Table row padding | 10px 16-20px (`py-2.5 px-4` to `px-5`) |
| Section gap | 24-32px (`mb-6` to `mb-8`) |
| Grid gap | 12-16px (`gap-3` to `gap-4`) |

### Page Anatomy (v2)
```
┌─ Top Nav (56px sticky) ──────────────────────────────┐
│ [Logo] Explorer  Network  Assets  Contracts  [Search] │
├──────────────────────────────────────────────────────┤
│                                                      │
│  Title + Badges                                      │
│  UPPERCASE LABEL                                     │
│  $58,247.82          ← Hero number (text-4xl)        │
│  XLM 124,500 · Trustlines 12 · Signers 1            │
│                                                      │
│  All  Activity  Soroban  Signers   ← Underline tabs  │
│  ─────────────────────────────────                   │
│                                                      │
│  ┌─ Table ──────────────────────┐  ← Primary content │
│  │ Asset    Type  Balance  USD  │                    │
│  │ XLM      Native 124,500 ... │                    │
│  └──────────────────────────────┘                    │
│                                                      │
│  Prism by Obsrvr            · Operational            │
└──────────────────────────────────────────────────────┘
```

---

## 5. Components

### Navigation
- **Top nav**: 56px sticky, `bg-white/95 backdrop-blur-md`, border-b border-gray-200.
- **Active state**: 2px bottom border in `#111`, text-gray-900. Hover: scaleX transition, 150ms ease.
- **Search bar**: Visible on all pages except home (which has hero search). `w-80 lg:w-96`, `rounded-xl`, `border-gray-300` with `ring-1`. `Ctrl+K` / `Cmd+K` focuses search or navigates to `/search`.
- **Footer**: Minimal single line — "Prism by Obsrvr" left, status dot right. No links, no background.
- **Network selector**: Dropdown with mainnet/testnet/futurenet, colored dots (emerald/amber/violet), persists via cookie.

### The 3-Tier Badge System
Badges are indicators, not buttons. Three tiers with distinct shapes:

| Tier | Component | Shape | Size | Ring | Use |
|------|-----------|-------|------|------|-----|
| **Status** | `@components.Badge(label, color)` | `rounded-full` | `text-2xs font-semibold` | `ring-1 ring-{color}-200` | Entity status: Validating, Success, Failed, Funded |
| **Type** | `@components.TypeBadge(label, color)` | `rounded` | `text-2xs font-bold` | `ring-1 ring-{color}-200/60` | Operation/entity types in tables: Contract Call, Payment, DEX |
| **Meta** | `@components.MetaBadge(label)` | `rounded` | `text-2xs font-semibold` | `ring-1 ring-gray-200` | Neutral metadata: Tier 1, Archive Publisher, SEP-41 |

**Rule**: Never create page-local badge templates. All badges route through these three shared components.

### Copy-to-Clipboard
The single most-used interaction in the explorer. Three states:

1. **Default**: Clipboard icon in `text-gray-300`
2. **Hover**: Icon darkens to `text-gray-600`
3. **Copied** (1.5s): Green checkmark in `text-emerald-500`, then reverts

Components: `@components.CopyButton(fullText)` for standalone, `@components.TruncatedAddress(fullAddr, href)` for linked addresses with copy.

### Tables
All data tables use semantic `<table>` elements with:
- **Header**: `text-2xs font-mono text-gray-400 uppercase tracking-widest bg-gray-50/80`
- **Rows**: `divide-y divide-gray-100`, hover `bg-gray-50/50`
- **Responsive**: Hide secondary columns at `md:` / `lg:` breakpoints using `hidden md:table-cell`

### Tabs
Underline-style, not pill-in-tray:
```
All          Classic          Soroban (SEP-41)
──────────
```
- Active: `border-b-2 border-gray-900 pb-3 text-gray-900 font-semibold`
- Inactive: `pb-3.5 text-gray-400 hover:text-gray-900`
- Maximum 5 tabs per page
- Tabs sit directly above the content area with `border-b border-gray-200`

### Buttons
| Type | Style | Use |
|------|-------|-----|
| Primary | `bg-gray-900 text-white rounded-lg` | CTAs: "View Full Details", "Got it" |
| Secondary | `border border-gray-200 bg-white rounded-lg` | Actions: "Share", "Export" |
| Pagination active | `bg-gray-900 text-white rounded-lg` | Current page |
| Pagination inactive | `border border-gray-200 bg-white rounded-lg` | Other pages |

### Charts
- **Bar charts** (validator 30D): CSS-only, flex container with `gap-[3px]`, bars are `flex-1 rounded-t-sm`. Healthy = `bg-emerald-500`, dips = `bg-amber-400`, rejected = `bg-red-400`.
- **Area charts** (network throughput): Inline SVG with fill at 6% opacity and stroke at full color.
- **Sparklines**: Server-rendered SVG via `@components.MiniSparkline(points, width, height, color)`.

---

## 6. Patterns

### Page Headers
Every page follows the same header structure. The header answers three questions in order:
1. **What am I looking at?** → Title + badges
2. **What's the key number?** → Hero number at `text-4xl font-bold tabular`
3. **What's the context?** → Inline stats with dot separators

#### Anatomy
```
Line 1: Title (text-2xl font-bold) + Status Badge + Type Badges
Line 2: Small uppercase label (text-2xs text-gray-400 uppercase tracking-wider)
Line 3: Hero number (text-4xl font-bold tabular)
Line 4: Inline stats (text-sm text-gray-500, values in font-semibold text-gray-900)
         separated by · (text-gray-300)
Then:   Underline tabs
```

#### Hero Numbers by Page
| Page | Hero Number | Question Answered |
|------|-------------|-------------------|
| Account | `$58,247.82` | "How much is in this account?" |
| Assets | `$24.8M` | "How much volume is moving?" |
| Network | `61,504,113` | "What ledger are we on?" |
| Events | `2,847` | "How many events matched?" |
| Smart Account | `$87,204.51` | "What's the treasury balance?" |
| Ledger Detail | `5,104,938` | "Which ledger?" (+ Etherscan-style key-value below) |
| Tx Receipt | *Summary text* | "What happened?" (human-readable, not a number) |

#### Rules
- **No stat card grids** — Deprecated in v2. They fragment attention across equal-weight metrics. Elevate one metric as the hero, push the rest to inline stats.
- **Two visual zones only** — Header (identity + hero + stats + tabs) → Content (tables/data). No intermediate dashboard layers.
- **Key-value overview for detail pages** — Ledger and transaction use Etherscan-inspired rows: fixed-width label column (`w-32`), gap to value, dot separators for secondary context.

### Data Display Rows
Two interaction modes:

| Mode | Element | Hover | Use |
|------|---------|-------|-----|
| **Navigable** | `<a>` tag | `hover:bg-gray-50/50` + right chevron | Entity lists: transactions, ledgers, contracts, assets, search results |
| **Informational** | `<div>` with `.detail-row` | Subtle bg change | Data tables with cell-level links: ledger detail txns, network ledgers, key-value pairs |

**Rule**: Never add `cursor-pointer` to a `<div>` that doesn't navigate. If the whole row is clickable, wrap it in `<a>`.

### Slide-out Panels
- **Width**: 480px fixed, slides from right
- **Backdrop**: `bg-black/8 backdrop-blur-[1px]`
- **Header**: Sticky, Back + Close buttons
- **htmx**: `hx-get="/assets/{code}/preview" hx-target="#slideout" hx-swap="innerHTML"`
- **CTA**: Full-width dark button at bottom — "View Full Details"

#### When to Use
| Pattern | Context | Examples |
|---------|---------|----------|
| **Slide-out** | Browsing from a list — show data the table CAN'T | Asset preview (issuer trust, Soroban integration, distribution, DEX pairs), validator preview |
| **Full page** | Committed inspection — primary content needs full width | Ledger detail, transaction receipt, account portfolio |

**Critical rule**: If the slide-out would just repeat the table data, don't build one. The panel must reveal *new* information.

---

## 7. Animation

Motion is fast, subtle, and purposeful:

| Animation | Properties | Duration | Use |
|-----------|------------|----------|-----|
| Fade up (page load) | `opacity 0→1, translateY 6px→0` | 300ms, 80ms stagger | All page content |
| Nav underline | `scaleX 0→1` | 150ms ease | Tab/nav hover |
| Card hover | `translateY(-2px)` + shadow | 150ms ease | Quick access cards |
| Pulse ring | `box-shadow 0→7px` | 2s infinite | Live network dot |
| Row hover | `background-color` | 80ms | Table rows |
| Copy feedback | Icon swap (clipboard → checkmark) | 1.5s timeout | CopyButton |

**Rule**: No bouncing, no overshooting, no decorative animation in data-dense views. The only continuous animation is the pulsing live dot.

---

## 8. Status & Feedback

Three semantic states with consistent surface treatment:

| State | Border | Background | Text | Threshold |
|-------|--------|------------|------|-----------|
| Healthy / Success | `border-emerald-200/60` | `bg-emerald-50/20` | `text-emerald-600/700` | Values >= 99.9% |
| Warning / Caution | `border-amber-200/60` | `bg-amber-50/20` | `text-amber-600/700` | Values < 99.9% or flagged |
| Error / Failed | `border-red-200/60` | `bg-red-50/20` | `text-red-600/700` | Values < 99% or failed tx |

**Note**: These container-level color treatments are used sparingly — for threshold stat cards and alert banners only. Normal data cards are always white with gray borders. Color comes from the *data values*, not the *containers*.

---

## 9. Do's and Don'ts

### Do
- **Do** use `tabular-nums` for every numerical value in tables and stats
- **Do** use the 480px slide-out to keep users in their browsing flow
- **Do** use a single hero number per page to answer the first question
- **Do** use Emerald for anything indicating a healthy/live network state
- **Do** use Violet exclusively for Soroban/smart contract data
- **Do** use semantic `<table>` elements for all tabular data
- **Do** use dot separators (`·` in `text-gray-300`) between inline stats

### Don't
- **Don't** use 6-column stat card grids — deprecated in v2
- **Don't** use violet-tinted containers — color goes on values, not backgrounds
- **Don't** use `cursor-pointer` on `<div>` elements — use `<a>` tags for navigable rows
- **Don't** create page-local badge helpers — use shared `Badge`, `TypeBadge`, `MetaBadge`
- **Don't** use blue for links — use Emerald to maintain brand signature
- **Don't** use pure black (`#000000`) for text — use `text-text-primary`
- **Don't** use hardcoded neutral grays (`bg-white`, `text-gray-900`, `border-gray-200`) — use semantic tokens (`bg-surface-card`, `text-text-primary`, `border-border-default`)
- **Don't** build a slide-out that repeats table data — it must reveal new information
