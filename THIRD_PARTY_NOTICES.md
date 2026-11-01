# Third-party notices

ImagePort is licensed under Apache-2.0. Its dependencies retain their own licenses.
The following are the primary directly used components; this is not a substitute
for a complete dependency license inventory for a particular release artifact.

| Component | License | Source |
| --- | --- | --- |
| Noto Sans CJK SC 2.004 (bundled regular/bold fonts) | OFL-1.1; [full license](assets/fonts/OFL.txt) | https://github.com/notofonts/noto-cjk/tree/Sans2.004 |
| Fyne | BSD-3-Clause | https://github.com/fyne-io/fyne |
| regclient | Apache-2.0 | https://github.com/regclient/regclient |
| Moby Go SDK | Apache-2.0 | https://github.com/moby/moby |
| Docker CLI context/config libraries | Apache-2.0 | https://github.com/docker/cli |
| distribution/reference | Apache-2.0 | https://github.com/distribution/reference |
| containerd/errdefs | Apache-2.0 | https://github.com/containerd/errdefs |
| go-keyring | MIT | https://github.com/zalando/go-keyring |
| gofrs/flock | BSD-3-Clause | https://github.com/gofrs/flock |
| go-i18n | MIT | https://github.com/nicksnyder/go-i18n |
| golang.org/x/text | BSD-3-Clause | https://go.googlesource.com/text |
| opencontainers/go-digest | Apache-2.0 | https://github.com/opencontainers/go-digest |
| modernc.org/sqlite | BSD-3-Clause | https://gitlab.com/cznic/sqlite |
| olareg (test registry) | Apache-2.0 | https://github.com/olareg/olareg |

Review transitive dependencies and redistribute required notices when producing
release packages. Go module versions and checksums are recorded in `go.mod` and
`go.sum`. Do not imply that upstream maintainers endorse ImagePort.
