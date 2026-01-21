name: Feature request
description: Propose a new feature or improvement
title: "[Feature] "
labels: ["feature"]
body:
  - type: textarea
    id: problem
    attributes:
      label: Problem / Motivation
      placeholder: What problem is this solving?
    validations:
      required: true
  - type: textarea
    id: proposal
    attributes:
      label: Proposed Solution
    validations:
      required: true
  - type: textarea
    id: impact
    attributes:
      label: Impact / Risk
