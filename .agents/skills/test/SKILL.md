---
name: test
description: "The TDD discipline: seams, the red → green loop, and the anti-patterns to delete. Use when you write or change tests, implement against acceptance criteria, or decide where a test belongs, which seam it must target, and whether it is worth keeping."
---

# Test

TDD is the red → green loop. This skill tells what a good test is, where tests go, the anti-patterns, and the rules of the loop. Each section applies on each cycle. Read the sections before the loop and during the loop, not after it.

## What a good test is

A test verifies behavior through a public interface, never through implementation details. A good test reads like a specification. Example: "user can check out with a valid cart" states exactly which capability exists. A good test survives refactors, because it does not depend on internal structure. Read `docs/GLOSSARY.md` and `docs/adr`, so that test names and interface vocabulary agree with the project's domain language. Obey the ADRs in the area that you touch.

## Seams — where tests go

A **seam** is the public boundary that you test at: the interface where behavior is observable from outside.

**Test only at seams that were agreed before.** The spec's Testing Decisions section names them. If the spec does not name them, agree the seams with the user before the first test. In a AFK session, take the outermost seam that can observe the acceptance criteria, and record the selection in the issue's notes. Agreement on seams before the work directs the test effort to critical paths and complex logic, not to each edge case.

## Rules of the loop

- **Red before green.** Write the failing test first. See it fail. Then write only enough code to make it pass. Do not add speculative features. Do not write tests for behavior that does not exist yet.
- **One slice at a time.** One seam, one test, one minimal implementation in each cycle. Each test is a tracer bullet that reacts to what the last cycle taught you.
- **Refactor work is not part of the loop.** The red → green cycle only makes the test pass. Improvements come later as findings from the `review-code` skill, and they flow back as issues.

## Anti-patterns — delete them when you see them

- **Implementation-coupled** — the test mocks internal collaborators, tests private functions, or verifies through a side channel (for example, a database query instead of the interface). The signal: the test breaks during a refactor while behavior is unchanged.
- **Tautological** — the assertion computes the expected value the same way that the code does (`expect(add(a, b)).toBe(a + b)`). The test passes by construction. Get the expected values from an independent source of truth: a known-good literal, a worked example, or the spec.
- **Horizontal slices** — you write all the tests first, then all the implementation. Tests written in bulk verify _imagined_ behavior, and they lock the structure before the implementation teaches you. Work in vertical slices: one test → one implementation → again.
