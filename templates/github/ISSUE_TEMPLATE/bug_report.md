name: Bug report
description: Report a reproducible bug
title: "[Bug] "
labels: ["bug"]
body:
  - type: textarea
    id: summary
    attributes:
      label: Summary
      description: What is broken and what should happen?
      placeholder: Clear, concise summary.
    validations:
      required: true
  - type: textarea
    id: steps
    attributes:
      label: Steps to Reproduce
      description: Exact steps to reproduce.
      placeholder: "1) ... 2) ..."
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: Expected Behavior
    validations:
      required: true
  - type: textarea
    id: actual
    attributes:
      label: Actual Behavior
    validations:
      required: true
  - type: input
    id: version
    attributes:
      label: Version / Commit
