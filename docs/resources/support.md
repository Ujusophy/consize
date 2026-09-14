# Support

Where to go depends on what you've got.

| Situation | Where |
|---|---|
| Something's broken and you've checked [Troubleshooting](troubleshooting.md) | [Open a GitHub issue](https://github.com/consize-oss/consize/issues/new) |
| A "how do I" or "why does X work this way" question, and [FAQ](faq.md) doesn't cover it | [Start a GitHub Discussion](https://github.com/consize-oss/consize/discussions) |
| Proposing a larger change (new provider, safety model change, new component) | [Start a GitHub Discussion](https://github.com/consize-oss/consize/discussions) first, see [Decisions](../contributing/decisions.md) |
| A security vulnerability | See [SECURITY.md](https://github.com/consize-oss/consize/blob/main/SECURITY.md), please don't file these as public issues |
| You want to contribute a fix or feature | See [Contributing](../contributing/index.md) |

## Before you ask

If it's a bug report, please include:

* Your Consize version (the Helm chart `--version`, or `git rev-parse HEAD` if you built from source)
* The output of `kubectl -n consize-system get pods`
* What you expected vs. what happened

## Next steps

* [Troubleshooting](troubleshooting.md)
* [FAQ](faq.md)
* [Contributing](../contributing/index.md)