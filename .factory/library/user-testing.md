# User Testing Guide

## Mission Context

This mission is a **documentation/analysis mission** producing a comparative report between `main` and `gem` branches. No services or applications are started.

## Testing Surfaces

### Document Validation (manual-review)

For analysis missions, assertions use `manual-review` tool. Validation involves:
1. Reading the produced document
2. Verifying required sections exist
3. Checking quantitative and qualitative justifications
4. Ensuring consistency between sections
5. Verifying traceability (code citations, file references)

## Validation Concurrency

This mission has **no concurrency constraints** - document validation is done by a single validator reading the report file.

## Flow Validator Guidance: Document Validation

**Isolation boundary:** None required - read-only document analysis.

**Constraints:**
- Only read files, do not modify
- Check document structure matches validation contract requirements
- Verify each conclusion has supporting evidence (file paths, line numbers, code snippets)

**Tools:**
- Read tool for document content
- Grep/Glob for code verification if needed

## Previous Runs

| Milestone | Round | Status | Notes |
|-----------|-------|--------|-------|
| synthesis | 1 | pass | All 12 assertions validated for code, functionality, tests, docs |
| report | 1 | pass | All 4 assertions validated for recommendation, plan, coherence, traceability |
