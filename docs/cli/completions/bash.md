
# Bash completions

## Adding completions to your console's profile

### Bash

Execute the following command to add the completions to your profile:

```sh
echo 'source <(c8y completion bash)' >> ~/.bashrc
```

### zsh

Execute the following command to add the completions to your profile:

```sh
echo 'source <(c8y completion zsh)' >> ~/.zshrc
```

To enable reverse cyling through completion options, add the following to your profile
```sh
bindkey '^[[Z' reverse-menu-complete
```

#### MacOS Troubleshooting

If the completions aren't working in the zsh, then try following the instructions here to resolve the issue.

https://scriptingosx.com/2019/07/moving-to-zsh-part-5-completions/#

