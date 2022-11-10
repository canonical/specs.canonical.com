# [specs.canonical.com](https://specs.canonical.com/)

A simple website to visualise Canonical specification documents stored in Google Drive. It allows filtering by different categories (status, team, etc.) as well as seeing a preview of the document. 

## Bugs and issues

If you have found a bug on the site or have an idea for a new feature, feel free to create a new issue, or suggest a fix by creating a pull request.


## Local development

The simplest way to run the site locally is using [dotrun](https://github.com/canonical/dotrun).

You'll need service account credentials to be able to access the spreadsheet that contains the specs metadata. Ask the Web & Design team about them.

In a `.env.local` file add the credentials:

```
PRIVATE_KEY_ID=...
PRIVATE_KEY=...
```

Run the project with:

```
dotrun
```

Once the server has started, you can visit http://127.0.0.1:8104 in your browser.

