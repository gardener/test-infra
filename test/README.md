# Integration tests for testmachinery controller

## Overview Execution procedure

- make install: install prerequisites of the tm controller which includes argo, minio and the testrun crd
- make install-controller: install the testmachinery-controller with necessary rbac rules and validation webhook
- run integration tests
- make remove-controller
- make remove


## Test validation webhook

- validation failing:
  - [x] location:
    - [x] missing locations
    - [x] local testrun
  - [x] testflow:
    - [x] no testdefs found or defined
    - [x] label with no testdefs
  - [x] invalid kubeconfig synatx (no connection test)

- validating succeed
  - [x] no testruns in onExit

## Controller tests

- [x] testflow with name
- [x] testflow with label
- [ ] onExit
  - [x] success condition (called when tesrun succeeds, not called when testrun fails)
  - [x] failing condition
  - [ ] always
- [x] TTL termination after workflow is completed
- [x] TTL should wait until workflow has finished
- [x] garbage collection on delete

## Testrunner tests

- [x] Test result summary as elastic search bulk request.
- [x] Test result summary with exportted artifacts of specific test steps.
