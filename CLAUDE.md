# CLUADE.md — Workspace guide for Claude Code

This repository is a **React UI/design-system style** frontend codebase built around:
- **React**
- **MUI (Material UI)**
- **react-hook-form** for forms
- **zod** (+ `@hookform/resolvers/zod`) for validation
- Project path aliases like `src/...`

Use this doc to understand structure, conventions, and how to make safe changes.

---

## 1) Project layout (high level)

Common directories:
- `src/sections/**`  
  Feature-level UI sections/pages. Example: `src/sections/client/client-create-edit-form.jsx`
- `src/components/**`  
  Reusable UI components and wrappers. Includes form helpers.
- `src/routes/**`  
  Route helpers (`paths`) and routing hooks (`useRouter`).
- `src/locales/**`  
  i18n; use `useTranslate()` and translation keys instead of hardcoded strings where possible.

---

## 2) How forms are built here (important)

Forms typically use:
- `useForm()` from `react-hook-form`
- A **zod schema** (example: `ClientCreateSchema`) and `zodResolver(schema)`
- Local `defaultValues`
- Wrapper components from `src/components/hook-form`:
  - `<Form methods={methods} onSubmit={...}>`
  - `<Field.Text />`, `<Field.Select />`, `<Field.Phone />`, `<Field.UploadAvatar />`, etc.

### When adding or changing a field
Update all of:
1. Zod schema (required/optional/nullable + error messages)
2. `defaultValues`
3. UI field component (`<Field.* name="..." />`)
4. Submission payload mapping (if transforming data before sending to API)

### Validation conventions
- Prefer zod constraints (`min(1)`, `.email()`, etc.)
- Use `schemaHelper.nullableInput(...)` when form values can be `null` but still required logically.
- Keep schema messages consistent and user-facing.

---

## 3) i18n / text conventions

- Prefer `const { t } = useTranslate()` and translation keys like:
  - `t('client-management.form-data.detail.title')`
- Avoid introducing new hardcoded strings unless there is no translation key yet.
- If you add keys, add them to the locale files under `src/locales/**`.

---

## 4) UI conventions

- Use MUI layout primitives (`Stack`, `Grid`, `Box`, `Card`) consistently.
- Keep responsive sizing via `sx` and `Grid size={{ xs, sm, md }}` patterns used in existing files.
- Keep icons consistent (FontAwesome is used in some sections).

---

## 5) Running locally (verify scripts in package.json)

From repo root:

```bash
# install deps (choose the one used by the repo)
npm install
# or: yarn
# or: pnpm install

# dev server
npm run dev

# build
npm run build

# tests (if configured)
npm test

# lint (if configured)
npm run lint
```

If any command is missing, check `package.json` for the correct script names.

---

## 6) Common integration points

- Routes:
  - `src/routes/paths` provides route constants (e.g. `paths.dashboard.user.list`)
  - `src/routes/hooks` provides navigation (e.g. `useRouter()`)

- Notifications:
  - `toast` from `src/components/snackbar`

- Form wrappers:
  - `Form`, `Field`, `schemaHelper` from `src/components/hook-form`

---

## 7) Guardrails for automated edits (Claude Code)

When making changes:
- Keep changes **small and localized**.
- Do not add new dependencies without an explicit request.
- Preserve existing patterns (file structure, naming, validation style).
- Avoid mixing unrelated refactors with functional changes.
- If you modify a form:
  - ensure schema + defaultValues + UI remain consistent
  - ensure submit still works (no missing required values)

---

## 8) Current reference file (example)

`src/sections/client/client-create-edit-form.jsx`
- Uses zod schema + `react-hook-form`
- Uses MUI layout (2-column section + additional card)
- Uses `Field.*` components for consistent form bindings
- Maintains local UI state for subscription module selection (`selectedModules`)

Note: if subscription modules must be submitted, they should be added to:
- schema (`modules: zod.array(zod.string())...`)
- defaultValues (`modules: []`)
- and the form value should be controlled via RHF (not only `useState`), unless intentionally UI-only.

---