---
title: Voyager Completion
menu:
  docs_{{ .version }}:
    identifier: voyager-completion
    name: Voyager Completion
    parent: reference-cli
menu_name: docs_{{ .version }}
section_menu_id: reference
---
## voyager completion

Generate completion script

### Synopsis

To load completions:

Bash:

$ source <(kubectl-voyager completion bash)

# To load completions for each session, execute once:
Linux:
  $ kubectl-voyager completion bash > /etc/bash_completion.d/kubectl-voyager
MacOS:
  $ kubectl-voyager completion bash > /usr/local/etc/bash_completion.d/kubectl-voyager

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ kubectl-voyager completion zsh > "${fpath[1]}/_kubectl-voyager"

# You will need to start a new shell for this setup to take effect.

Fish:

$ kubectl-voyager completion fish | source

# To load completions for each session, execute once:
$ kubectl-voyager completion fish > ~/.config/fish/completions/kubectl-voyager.fish


```
voyager completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [voyager](/docs/reference/cli/voyager.md)	 - Voyager cli by AppsCode

