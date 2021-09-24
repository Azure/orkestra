# Contributing to Orkestra

Welcome to Orkestra! We would love to accept your patches and contributions to this project.

Here is how you can help:

- Report or fix bugs.
- Add or propose new features.
- Improve our documentation.

## Resources

- Website: https://azure.github.io/orkestra/
- Documentation: https://azure.github.io/orkestra/api.html
- Developer's Guide: https://azure.github.io/orkestra/developers.html
- Discussion forum: https://github.com/Azure/orkestra/discussions
- Azure Orkestra Slack: Join the Azure Orkestra [Slack](https://join.slack.com/t/azureorkestra/shared_invite/zt-rowzrite-Hm_eaih4GyjjZXWftuoqPQ)

## Pull Request Checklist

Before sending your pull requests, make sure you do the following:

- Read this contributing guide.
- Read the [Code of Conduct][code-conduct-link].
- Run the [tests](#running-tests).
- Run `make prepare-for-pr`. For details, see [Prepare Code for PR](#prepare-code-for-pr) section.

## How to become a contributor

### Contributor License Agreement

Most contributions to this project require you to agree to a Contributor License Agreement (CLA) declaring that you have the right to, and do, grant us the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

### Finding something to work on

If you want to write some code, but don't know where to start or what you might want to do, take a look at the [Good First Issue][good-issue] label.

### Developing Orkestra

Follow the [Developer's Guide][dev-guide] for a full set of instructions to get started with building, running, and debugging Orkestra.

### Running tests

For a full set of instructions to run tests, follow the testing & debugging section of [Developer's Guide][dev-guide-tests]. For some tips, you can run tests with the following `make` target.

```shell
make clean && make dev && make test
```

### Prepare Code for PR

Before submitting a PR, run the following `make` target.

```shell
make prepare-for-pr
```

This will perform following checks:
- Examine Go source code and report suspicious constructs.
- Format Go source code.
- Update API docs if applicable.

## Security

For instructions on reporting security issues and bugs, please see [security][security-link] guide.


## Support

For questions about building, running, or troubleshooting, start with the [Developer's Guide][dev-guide], and work your way through the process that we've outlined. If that doesn't answer your question(s), try to post on [Discussion][discussion-link] tab or if you think you found a bug, please file an [issue][issue-link].

If you still have question(s), join the Azure Orkestra [Slack](https://join.slack.com/t/azureorkestra/shared_invite/zt-rowzrite-Hm_eaih4GyjjZXWftuoqPQ) and someone will help you get answer(s) to your question(s).

[dev-guide]: https://azure.github.io/orkestra/developers.html
[dev-guide-tests]: https://azure.github.io/orkestra/developers.html#testing--debugging
[code-conduct-link]: https://github.com/Azure/orkestra/blob/main/CODE_OF_CONDUCT.md
[security-link]: https://github.com/Azure/orkestra/blob/main/SECURITY.md
[good-issue]: https://github.com/Azure/orkestra/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22
[discussion-link]: https://github.com/Azure/orkestra/discussions
[issue-link]: https://github.com/Azure/orkestra/issues/new/choose
