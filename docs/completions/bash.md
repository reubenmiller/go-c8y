
# Bash completions

## Requirements
* bash-completions. If it is not installed then the following error will appear `bash: _get_comp_words_by_ref: command not found warning`

Bash completions can be installed using one of the following:

**CentOS**

```sh
yum install bash-completion bash-completion-extras
```

**MacOS**

```sh
brew install bash-completion
```

**Note:** You need to start a new bash session before the bash add-ons are activated

## Instructions

Generating bash completions

```sh
c8y completions bash > .c8y.sh

# Add the following to your .bash_profile
source .c8y.sh
```
