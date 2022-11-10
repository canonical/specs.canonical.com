# [specs.canonical.com](https://specs.canonical.com/)

A simple website to visualise Canonical specification documents stored in Google Drive. It allows filtering by different categories (status, team, etc.) as well as seeing a preview of the document. 

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

