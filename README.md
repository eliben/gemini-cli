# gemini-cli

`gemini-cli` is a simple yet powerful command-line interface for Google's Gemini LLMs,
written in Go. It includes tools for chatting with these models and
generating / comparing embeddings, with powerful SQLite storage and analysis capabilities.

## Installing

Install `gemini-cli` on your machine with:

```
$ go install github.com/eliben/gemini-cli@latest
```

You can then invoke `gemini-cli help` to verify it's properly installed and found.

## Usage

All `gemini-cli` invocations require the API key for https://ai.google.dev/ to be provided,
either via the `--key` flag or an environment variable called `GEMINI_API_KEY`. From here
one, all examples assume environment variable was set earlier to a valid key.

`gemini-cli` has a nested tree of subcommands to perform various tasks. You can always
run `gemini-cli help <command> [subcommand]...` to get usage; e.g. `gemini-cli help chat`
or `gemini-cli help embed similar`. The printed help information will describe every
subcommand and its flags.

This guide will discuss some of the more common use cases.

### `prompt` - single prompts

The `prompt` command allows one to send queries consisting of text or images to the LLM.
This is a single-shot interaction; the LLM will have no memory of previous prompts (see
that `chat` command for with-memory interactions).

```
gemini-cli prompt <prompt or '-'>... [flags]
```

The prompt can be provided as a sequence of parts, each one a command-line argument.

The arguments are sent as a sequence to the model in the order provided.
If `--system` is provided, it's prepended to the other arguments. An argument
can be some quoted text, a name of an image file on the local filesystem or
a URL pointing directly to an image file online. A special argument with
the value `-` instructs the tool to read this prompt part from standard input.
It can only appear once in a single invocation.

Some examples:





