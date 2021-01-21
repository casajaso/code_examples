## Contribute to AWS Okta CLI

Thank you for your interest in contributing to the **AWS Okta CLI**. This guide details how
to contribute to the project in a way that is easy for everyone.

## Security vulnerability disclosure

Please report suspected security vulnerabilities in private to
`cloudengineering@nordstrom.com` (cc' `cloudsecurity@nordstrom.com`). Please do **NOT** create publicly viewable issues for suspected security
vulnerabilities.

## Code of conduct

As contributors and maintainers of this project, we pledge to respect all
people who contribute through reporting issues, posting feature requests,
updating documentation, submitting pull requests or patches, and other
activities.

Project maintainers have the right and responsibility to remove, edit, or
reject comments, commits, code, wiki edits, issues, and other contributions
that are not aligned to this Code of Conduct.

## I want to contribute!

If you want to contribute to the AWS Okta CLI, [Gitlab issues](https://gitlab.nordstrom.com/public-cloud/aws-okta/issues)
are a great place to start. If you have any questions or need help, please email `cloudengineering@nordstrom.com`. Thanks for your contribution!

## Contribution Flow

When contributing to the **AWS Okta CLI**, your merge request is subject to review by merge request maintainers of a particular specialty.

When you submit code, we really want it to get merged, but there will be times when it will not be merged.

When maintainers are reading through a merge request they may request guidance from other maintainers. If merge request maintainers conclude that the code should not be merged, our reasons will be fully disclosed. If it has been decided that the code quality is not up to Nordstrom's standards, the merge request maintainer will refer the author to our docs and code style guides, and provide some guidance.

Sometimes style guides will be followed but the code will lack structural integrity, or the maintainer will have reservations about the code’s overall quality. When there is a reservation the maintainer will inform the author and provide some guidance.  The author may then choose to update the merge request. Once the merge request has been updated and reassigned to the maintainer, they will review the code again. Once the code has been resubmitted any number of times, the maintainer may choose to close the merge request with a summary of why it will not be merged, as well as some guidance. If the merge request is closed the maintainer will be open to discussion as to how to improve the code so it can be approved in the future.

The Cloud Engineering team will do its best to review community contributions as quickly as possible.

When submitting code, you may feel that your contribution requires the aid of an external library. If your code includes an external library please provide a link to the library, as well as reasons for including it.

# Merge requests

We welcome merge requests with fixes and improvements to Cloud Engineering code, tests,
and/or documentation.

Please note that if an issue is marked for the current milestone either before
or while you are working on it, a team member may take over the merge request
in order to ensure the work is finished before the release date.

If you want to add a new feature that is not labeled it is best to first create
a feedback issue (if there isn't one already).

To start with **AWS Okta CLI** development, make sure you have [Go](https://golang.org/doc/install) installed.

## Merge Request guidelines

If you can, please submit a merge request with the fix or improvements
including tests. If you don't know how to fix the issue but can write a test
that exposes the issue we will accept that as well. In general bug fixes that
include a regression test are merged quickly while new features without proper
tests are least likely to receive timely feedback. The workflow to make a merge
request is as follows:

1. Fork the project into your personal space on gitLab.nordstrom.com
1. Create a feature branch, branch away from `master`
1. Write tests and code
1. If you have multiple commits please combine them into a few logically organized commits by squashing them.
1. Push the commit(s) to your fork
1. Submit a merge request (MR) to the `master` branch
   1. Your merge request needs at least 1 approval but feel free to require more.
   1. You don't have to select any approvers, but you can if you really want
      specific people to approve your merge request
1. The MR title should describe the change you want to make
1. The MR description should give a motive for your change and the method you used to achieve it.
   1. If you are contributing code, fill in the template already provided in the
      "Description" field.
   1. Mention the issue(s) your merge request solves, using the `Solves #XXX` or
      `Closes #XXX` syntax to auto-close the issue(s) once the merge request will
      be merged.
1. If you're allowed to, set a relevant milestone and labels
1. Be prepared to answer questions and incorporate feedback even if requests
   for this arrive weeks or months after your MR submission
   1. If a discussion has been addressed, select the "Resolve discussion" button
      beneath it to mark it resolved.
1. When writing commit messages please follow
   [these](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)
   [guidelines](http://chris.beams.io/posts/git-commit/).

Please keep the change in a single MR **as small as possible**. If you want to
contribute a large feature think very hard what the minimum viable change is.
Can you split the functionality? The increased reviewability of small MRs that leads to higher code quality is more important
to us than having a minimal commit log. The smaller an MR is the more likely it
is it will be merged (quickly). After that you can send more MRs to enhance it.

## Contribution acceptance criteria

1. The change is as small as possible
1. Include proper tests and make all tests pass (unless it contains a test
   exposing a bug in existing code).
1. If you suspect a failing CI build is unrelated to your contribution, you may
   try and restart the failing CI job or ask a developer to fix the
   aforementioned failing test.
1. Your MR initially contains a single commit (please use `git rebase -i` to
   squash commits)
1. Your changes can merge without problems (if not please rebase if you're the
   only one working on your feature branch, otherwise, merge `master`)
1. Does not break any existing functionality
1. Fixes one specific issue or implements one specific feature (do not combine
   things, send separate merge requests if needed)
1. Keeps the code base clean and well structured
1. Contains functionality we think other users will benefit from too
1. Changes after submitting the merge request should be in separate commits
   (no squashing).
1. The merge request meets the [definition of done](#definition-of-done).

## Definition of done

If you contribute to the **AWS Okta CLI**, please know that changes involve more than just
code. We have the following [definition of done][definition-of-done]. Please ensure you support
the feature you contribute through all of these steps.

1. Description explaining the relevancy
1. Working and clean code that is commented where needed
1. Unit, integration, and system tests that pass on the Gitlab CI server
1. Performance/scalability implications have been considered, addressed, and tested
1. Documented in the `/doc` directory
1. Changelog entry added, if necessary
1. Reviewed and any concerns are addressed
1. Merged by a project maintainer
1. Answers to questions radiated (in docs/wiki/support etc.)

If you add a dependency to the **AWS Okta CLI** (such as an operating system package), please
consider noting the applicability in your merge request.
