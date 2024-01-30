from webapp.google import Drive, Sheets
from webapp.spec import Spec
from webapp.settings import (
    TRACKER_SPREADSHEET_ID,
    TEAMS_FOLDER_ID,
    SPECS_SHEET_TITLE,
    TMP_SHEET_TITLE,
)


def update_sheet() -> None:
    """
    Get specs from Google Drive, parse the metadata on top of the document
    and write into a spreadsheet
    """
    drive = Drive()
    sheets = Sheets(spreadsheet_id=TRACKER_SPREADSHEET_ID)

    specs_sheet = sheets.get_sheet_by_title(SPECS_SHEET_TITLE)
    tmp_sheet = sheets.ensure_sheet_by_title(TMP_SHEET_TITLE)

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
            try:
                comments = drive.get_comments(
                    file_id=file["id"], fields=("resolved",)
                )
                open_comments = [c for c in comments if not c["resolved"]]

                parsed_doc = Spec(google_drive=drive, document_id=file["id"])
            except Exception as e:
                print(f"Unable to parse document: {file['name']}", e)
                continue

            row = [
                folder["name"],
                file["name"],
                file["id"],
                file["webViewLink"],
                parsed_doc.metadata.get("index"),
                parsed_doc.metadata.get("title"),
                parsed_doc.metadata.get("status"),
                ", ".join(parsed_doc.metadata.get("authors")),
                parsed_doc.metadata.get("type"),
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
