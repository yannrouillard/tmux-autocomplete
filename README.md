# tmux-autocomplete

tmux autocomplete tool offers a way to complete identifiers from current pane.

## Configuration

Modify `~/tmux.conf` to enable autocompletion:

```
bind-key -n <your-key> "tmux-autocomplete"
```

Pressing `<your-key>` will start completion mode based on the identifier which
is just before cursor.

Use arrow keys to navigate and select proper candidate.
