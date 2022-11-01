from webapp.google import Docs, Drive, Sheets
from webapp.contants import TRACKER_SPREADSHEET_ID, SPECS_SHEET_TITLE


TEAMS_FOLDER_ID = "19jxxVn_3n6ZAmFl3DReEVgZjxZnlky4X"
TMP_SHEET_TITLE = "Specs_tmp"


def _parse_top_table(document: dict) -> dict:
    """
    Parse the content of the table on top of the document,
    which contains spec metadata
    """
    allowed_keys = ("Index", "Title", "Status", "Authors", "Type", "Created")

    doc_content = document.get("body").get("content")
    table_metadata = {}
    for element in doc_content:
        if "table" not in element:
            continue

        table = element.get("table")
        for row in table.get("tableRows"):
            cells = row.get("tableCells")
            key = cells[0]["content"][0]["paragraph"]["elements"][0][
                "textRun"
            ]["content"].strip()

            if key not in allowed_keys:
                continue

            values = cells[1]["content"][0]["paragraph"]["elements"]

            if len(values) == 1:
                table_metadata[key] = values[0]["textRun"]["content"].strip()
            elif all(v.get("textRun") for v in values):
                # Some text appears as several items despite being clearly part
                # of the same word. In that case, join it in a single string
                table_metadata[key] = "".join(
                    v["textRun"]["content"] for v in values
                )
            else:
                # Generate a list of people
                table_metadata[key] = "".join(
                    v["person"]["personProperties"]["name"].strip()
                    for v in values
                    if v.get("person")
                )

        break

    return table_metadata


def update_sheet() -> None:
    """
    Get specs from Google Drive, parse the metadata on top of the document
    and write into a spreadsheet
    """
    drive = Drive()
    docs = Docs()
    sheets = Sheets(spreadsheet_id=TRACKER_SPREADSHEET_ID)

    specs_sheet = sheets.get_sheet_by_title(SPECS_SHEET_TITLE)
    tmp_sheet = sheets.get_sheet_by_title(TMP_SHEET_TITLE)

    sheets.clear(sheet_id=tmp_sheet["properties"]["sheetId"])

    # Add headers
    sheets.insert_rows(
        rows=[
            [
                "Folder name",
                "File name",
                "File ID",
                "File URL",
                "Index",
                "Title",
                "Status",
                "Authors",
                "Type",
                "Created",
                "Last updated",
                "Number of comments",
                "Number of open comments",
            ]
        ],
        range=TMP_SHEET_TITLE,
    )

    query_subfolders = (
        f"mimeType = 'application/vnd.google-apps.folder' "
        f"and '{TEAMS_FOLDER_ID}' in parents"
    )
    folders = drive.get_files(query=query_subfolders, fields=("id", "name"))

    for folder in folders:
        query_doc_files = (
            f"mimeType = 'application/vnd.google-apps.document' "
            f"and '{folder['id']}' in parents"
        )
        files = drive.get_files(
            query=query_doc_files,
            fields=(
                "id",
                "name",
                "createdTime",
                "modifiedTime",
                "webViewLink",
            ),
        )

        for file in files:
            comments = drive.get_comments(
                file_id=file["id"], fields=("resolved",)
            )
            open_comments = [c for c in comments if not c["resolved"]]

            document = docs.get_document(file["id"])
            table_metadata = _parse_top_table(document)

            if not table_metadata:
                print(f"Unable to parse document: {file['name']}")
                continue

            row = [
                folder["name"],
                file["name"],
                file["id"],
                file["webViewLink"],
                table_metadata.get("Index"),
                table_metadata.get("Title"),
                table_metadata.get("Status"),
                table_metadata.get("Authors"),
                table_metadata.get("Type"),
                file["createdTime"],
                file["modifiedTime"],
                len(comments),
                len(open_comments),
            ]
            sheets.insert_rows(
                rows=[row],
                range=TMP_SHEET_TITLE,
            )

    # Rename temporary file as the main one once it contains all the specs
    sheets.update_sheet_name(
        sheet_id=specs_sheet["properties"]["sheetId"], new_name="tmp"
    )
    sheets.update_sheet_name(
        sheet_id=tmp_sheet["properties"]["sheetId"],
        new_name=SPECS_SHEET_TITLE,
    )
    sheets.update_sheet_name(
        sheet_id=specs_sheet["properties"]["sheetId"],
        new_name=TMP_SHEET_TITLE,
    )
