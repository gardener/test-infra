# Prepare

Prepare step of the testmachinery that clones the specified repositories to the repo src path and creates specified directories.

Repositories and directories are specified by json config file with the form:
```json
{
  "directories": [ "/path1/repo", "/tmp/path2/" ],
  "repositories": [
    {
      "name": "unique name to identify repo",
      "url": "http clone url",
      "revision": "git branch or commit to checkout"
    }
  ]
}
```

Private repos are cloned by using the curl default `.netrc` file in the home directory of the user.
