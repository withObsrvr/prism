---
name: Prism
description: Soroban-first block explorer that turns blockchain data into understandable, trustworthy stories.
colors:
  surface-page: "#FAFAFA"
  surface-card: "#FFFFFF"
  surface-subtle: "#F3F4F6"
  border-default: "#E5E7EB"
  border-subtle: "#F3F4F6"
  text-primary: "#111111"
  text-strong: "#374151"
  text-body: "#6B7280"
  text-muted: "#9CA3AF"
  dark-surface-page: "#09090B"
  dark-surface-card: "#141414"
  dark-surface-subtle: "#1E1E1E"
  dark-border-default: "#2A2A2A"
  emerald-primary: "#10B981"
  emerald-link: "#047857"
  emerald-soft: "#ECFDF5"
  violet-soroban: "#7C3AED"
  violet-soft: "#F5F3FF"
  amber-warning: "#F59E0B"
  amber-soft: "#FFFBEB"
  red-error: "#EF4444"
  red-soft: "#FEF2F2"
  cyan-transfer: "#06B6D4"
typography:
  display:
    fontFamily: "Instrument Sans, SF Pro Display, system-ui, sans-serif"
    fontSize: "2.25rem"
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-0.025em"
  headline:
    fontFamily: "Instrument Sans, SF Pro Display, system-ui, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Instrument Sans, SF Pro Display, system-ui, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 700
    lineHeight: 1.35
  body:
    fontFamily: "Instrument Sans, SF Pro Display, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Instrument Sans, SF Pro Display, system-ui, sans-serif"
    fontSize: "0.625rem"
    fontWeight: 600
    lineHeight: 1rem
    letterSpacing: "0.08em"
  mono:
    fontFamily: "JetBrains Mono, ui-monospace, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  sm: "4px"
  md: "8px"
  lg: "12px"
  xl: "16px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  2xl: "24px"
  3xl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.text-primary}"
    textColor: "{colors.surface-card}"
    rounded: "{rounded.md}"
    padding: "10px 16px"
    typography: "{typography.body}"
  button-secondary:
    backgroundColor: "{colors.surface-card}"
    textColor: "{colors.text-strong}"
    rounded: "{rounded.md}"
    padding: "10px 16px"
    typography: "{typography.body}"
  input-search:
    backgroundColor: "{colors.surface-card}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.lg}"
    padding: "10px 16px"
    typography: "{typography.body}"
  badge-status:
    backgroundColor: "{colors.emerald-soft}"
    textColor: "{colors.emerald-link}"
    rounded: "{rounded.full}"
    padding: "2px 8px"
    typography: "{typography.label}"
  card-default:
    backgroundColor: "{colors.surface-card}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.lg}"
    padding: "16px"
---

# Design System: Prism

## 1. Overview

**Creative North Star: "The Financial Instrument"**

Prism should feel like a high-end financial instrument for blockchain investigation: calm, precise, and built for work. The interface serves people who arrive with hashes, accounts, ledgers, contracts, and questions. It must answer what happened first, then make the supporting evidence inspectable without friction.

The visual language is restrained product UI with functional density. Hierarchy comes from type, spacing, table structure, and semantic color, not from decorative panels or equal-weight dashboard blocks. A page has two main zones: a concise header that frames meaning, then content that supports inspection.

Prism explicitly rejects the raw hash dump, crypto-neon, generic explorer clone, and over-decorated AI dashboard. The product is allowed to be dense, but it must stay readable and trustworthy.

**Key Characteristics:**
- Restrained neutral surfaces with one primary interaction color.
- Semantic color only: status, entity type, interaction, or threshold.
- Mono for blockchain data, sans for interface language.
- Tables, key-value rows, tabs, and slide-outs over nested cards.
- Fast, purposeful motion that clarifies state.

## 2. Colors

The palette is a restrained neutral system with emerald as the signature interaction and health color, plus violet, amber, red, and cyan reserved for meaning.

### Primary
- **Signal Emerald**: Primary links, live states, healthy network indicators, copy success, positive deltas, and clickable blockchain data. All linked hashes, addresses, ledgers, and accounts use emerald rather than blue.

### Secondary
- **Soroban Violet**: Contract calls, smart contract badges, SAC assets, Soroban events, and function names. Violet belongs on values and labels, never as a decorative card wash.
- **Transfer Cyan**: Transfer events, lending badges, and narrow data-viz moments where emerald would imply health rather than category.

### Tertiary
- **Caution Amber**: Warning thresholds and concerning but non-critical states.
- **Failure Red**: Failed transactions, critical TTL, rejected storage entries, destructive deltas, and errors.

### Neutral
- **Paper Surface**: Page background and low-emphasis product canvas.
- **Card Surface**: Panels, nav, tables, dialogs, slide-outs, and content containers.
- **Subtle Surface**: Table headers, skeletons, hover states, and low-emphasis fills.
- **Evidence Border**: Card borders, table dividers, inputs, rings, and separation lines.
- **Primary Ink**: Headings, hero values, active tabs, and strong data.
- **Strong Gray**: Secondary headings, labels, and supporting values.
- **Body Gray**: Descriptions, helper copy, and low-priority metadata.
- **Muted Gray**: Placeholders, separators, timestamps, and background labels.

### Named Rules
**The Color Means Something Rule.** Every color must indicate status, entity type, interaction, or threshold. Decorative color is prohibited.

**The Emerald Link Rule.** Clickable blockchain data is emerald. Never use blue links in the explorer.

**The Violet Values Rule.** Violet identifies Soroban and smart contract data on values, badges, or text. Violet-tinted containers are forbidden.

## 3. Typography

**Display Font:** Instrument Sans, with SF Pro Display and system-ui fallbacks  
**Body Font:** Instrument Sans, with SF Pro Display and system-ui fallbacks  
**Label/Mono Font:** JetBrains Mono, with ui-monospace fallback

**Character:** Instrument Sans keeps the product human and direct. JetBrains Mono gives addresses, hashes, sequence numbers, fees, and amounts a technical rhythm that supports audit work.

### Hierarchy
- **Display** (700, 36px, 1.1): Hero numbers and the most important page value. Use sparingly, one per page when the page has a single key metric.
- **Headline** (700, 24px, 1.2): Page titles and major entity names.
- **Title** (700, 18px, 1.35): Section titles and panel headers.
- **Body** (400, 14px, 1.5): Descriptions, table body, explanations, and normal interface copy. Keep prose at 65 to 75ch when it becomes narrative.
- **Label** (600, 10px, uppercase, 0.08em tracking): Category labels, table headers, badge labels, and compact metadata.
- **Mono** (400 to 500, 10 to 14px, tabular): Addresses, hashes, ledger numbers, amounts, fees, sequences, public keys, and technical values.

### Named Rules
**The Mono Evidence Rule.** All blockchain data uses JetBrains Mono and tabular numerals. If a user might copy, compare, or audit it, it is mono.

**The One Hero Rule.** Most pages get one hero value or narrative summary. Do not create equal-weight metric grids that compete with the page's core answer.

## 4. Elevation

Prism uses a hybrid of tonal layering and shallow shadows. Borders and background contrast carry most structure. Shadows are reserved for interactive lift, search prominence, dialogs, slide-outs, and command surfaces.

### Shadow Vocabulary
- **Inline Shadow** (`0 1px 2px 0 rgba(0,0,0,0.05)`): Buttons and compact inline elements.
- **Card Shadow** (`0 1px 3px 0 rgba(0,0,0,0.06), 0 1px 2px -1px rgba(0,0,0,0.06)`): Default cards and key-value containers when they need slight separation.
- **Hover Shadow** (`0 4px 8px -2px rgba(0,0,0,0.06), 0 2px 4px -2px rgba(0,0,0,0.04)`): Quick access cards and interactive surfaces that lift by 2px.
- **Panel Shadow** (`0 12px 24px -4px rgba(0,0,0,0.08), 0 4px 8px -2px rgba(0,0,0,0.04)`): Search hero surfaces and slide-out panels.
- **Modal Shadow** (`0 24px 48px -8px rgba(0,0,0,0.10), 0 8px 16px -4px rgba(0,0,0,0.04)`): Dialogs and command palette surfaces.

### Named Rules
**The Flat Until Asked Rule.** Surfaces are flat or lightly bordered at rest. Shadows appear when a component needs focus, overlay hierarchy, or hover feedback.

**The No Murky Shadow Rule.** If a shadow makes the interface feel like a generic admin panel, remove it and use a border or tonal layer instead.

## 5. Components

### Buttons
- **Shape:** Quietly rounded rectangles (8px radius). Pagination and compact actions share the same radius.
- **Primary:** Primary actions use Primary Ink on Card Surface with 10px vertical and 16px horizontal padding, 14px semibold text, and a dark hover shift.
- **Hover / Focus:** Hover changes color only. Focus uses a visible emerald ring where appropriate. No bounce, no elastic motion.
- **Secondary / Ghost / Tertiary:** Secondary buttons use Card Surface, Evidence Border, and Strong Gray text. Ghost actions are text-first and reserved for nav or table utilities.

### Chips
- **Style:** Status badges are rounded-full with a soft semantic background, semantic text, and a 1px semantic ring. Type and meta badges use smaller rounded rectangles.
- **State:** Badges are indicators, not buttons. Filter chips may become interactive, but status, type, and meta badges remain non-clickable unless explicitly paired with a link affordance.

### Cards / Containers
- **Corner Style:** 12px for cards and key containers; 16px for hero search and larger preview panels.
- **Background:** Card Surface with Evidence Border. Use Subtle Surface for hover states, table headers, and skeletons.
- **Shadow Strategy:** Card Shadow only when a container must separate from the page. Data tables can rely on border and tonal layering.
- **Border:** 1px semantic border. Colored side-stripe borders are prohibited.
- **Internal Padding:** 16px to 20px for dense product containers; 24px to 28px for preview panels.

### Inputs / Fields
- **Style:** Card Surface, 1px Evidence Border, 8px to 16px radius depending on scale, Body text, Muted Gray placeholder.
- **Focus:** Emerald focus ring at 20 percent opacity with an emerald border shift. The search field can use a stronger 3px ring because it is a primary workflow object.
- **Error / Disabled:** Error uses Failure Red text or border plus text copy. Disabled uses Muted Gray text and Subtle Surface, never reduced opacity alone.

### Navigation
- **Style:** 56px sticky top nav with Card Surface, subtle translucency where supported, and a bottom border.
- **Default / Hover / Active:** Active nav and tabs use Primary Ink and a 2px underline. Hover uses a 150ms scaleX underline or text-color shift.
- **Mobile:** Preserve search and network selection. Collapse navigation structurally rather than shrinking type below readable sizes.

### Tables
- **Style:** Semantic tables with compact rows, uppercase mono headers, Subtle Surface headers, and 1px dividers.
- **Rows:** Navigable rows are anchors with subtle hover and a clear destination. Informational rows use a non-clickable container and no fake pointer cursor.
- **Responsive:** Hide secondary columns at breakpoints. Preserve the primary entity, status, and newest useful context.

### Slide-out Panels
- **Style:** 480px fixed-width right panel with Card Surface, border-left, and strong overlay shadow.
- **Use:** Browsing from a list where the panel reveals new information the table cannot show.
- **Rule:** If the panel repeats table data, do not build it.

## 6. Do's and Don'ts

### Do:
- **Do** use `tabular-nums` for every numerical value in tables, stats, and key-value rows.
- **Do** use emerald for clickable blockchain data and healthy or live network states.
- **Do** use violet exclusively for Soroban and smart contract meaning.
- **Do** lead each page with one clear answer: hero number, narrative summary, or entity identity.
- **Do** use the 480px slide-out when users are browsing a list and need richer preview information.
- **Do** use semantic tables for tabular data and key-value rows for detail pages.
- **Do** communicate status with text, icon, and shape as well as color to support color-blind users.
- **Do** respect reduced motion. Keep motion to state changes, feedback, loading, and reveal.

### Don't:
- **Don't** make Prism feel like a raw hash dump. Lead with meaning, then reveal evidence.
- **Don't** use crypto-neon, trading-screen glow, or futuristic effects as the visual identity.
- **Don't** create an over-decorated AI dashboard. No generic gradient panels, hero-metric cliché blocks, or decorative glass.
- **Don't** use blue links. Explorer links are emerald.
- **Don't** use violet-tinted containers. Violet belongs on Soroban values, labels, and badges.
- **Don't** use colored side-stripe borders for cards, list items, callouts, or alerts.
- **Don't** use 6-column stat card grids. They fragment attention and flatten hierarchy.
- **Don't** use `cursor-pointer` on non-navigating divs. If a whole row navigates, make it an anchor.
- **Don't** create page-local badge helpers. Route status, type, and meta badges through shared components.
- **Don't** hide raw evidence behind summaries. Every interpretation needs a path to proof.
