from typing import Dict, List
import datetime

import tenacity

from webapp.google import Drive, Sheets
from webapp.settings import (
    SPECS_SHEET_TITLE,
    TEAMS_FOLDER_ID,
    TMP_SHEET_TITLE,
    TRACKER_SPREADSHEET_ID,
)
from webapp.spec import Spec

try:
    from itertools import batched
except ImportError:
    from itertools import islice

    def batched(iterable, n):
        # batched('ABCDEFG', 3) --> ABC DEF G
        if n < 1:
            raise ValueError("n must be at least one")
        it = iter(iterable)
        while batch := tuple(islice(it, n)):
            yield batch


def _generate_spec_rows_for_folders(
    drive: Drive,
    folders: List[Dict],
    existing_specs: Dict[str, List],
):
    """
    Generate rows for the spreadsheet with the metadata of the specs
    found in the folders.

    :param drive: Drive instance
    :param folders: List of folders
    :param existing_specs: Dict of existing specs in the sheet
      key is file id, value is the row in the sheet
    """
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
        print(f"Found {len(files)} documents in {folder['name']}")

        for file_ in files:
            if file_["id"] in existing_specs:
                file_modification_date = datetime.datetime.fromisoformat(
                    file_["modifiedTime"][:-1]
                )
                row = existing_specs.get(file_["id"])
                previous_modified_date = row["values"][10]["formattedValue"]
                previous_modified_date = datetime.datetime.fromisoformat(
                    previous_modified_date[:-1]
                )
                if previous_modified_date >= file_modification_date:
                    # file has not been modified since last update
                    # use the same row as before
                    try:
                        new_row = [
                            value.get("formattedValue", "")
                            for value in row["values"]
                        ]
                        yield new_row
                    except KeyError as e:
                        print(
                            "something went wrong with file ", file_["name"], e
                        )

            try:
                comments = drive.get_comments(
                    file_id=file_["id"], fields=("resolved",)
                )
                open_comments = [c for c in comments if not c["resolved"]]
                parsed_doc = Spec(google_drive=drive, document_id=file_["id"])
            except Exception as e:
                print(f"Unable to parse document: {file_['name']}", e)
                continue

            row = [
                folder["name"],
                file_["name"],
                file_["id"],
                file_["webViewLink"],
                parsed_doc.metadata["index"],
                parsed_doc.metadata["title"],
                parsed_doc.metadata["status"],
                ", ".join(parsed_doc.metadata["authors"]),
                parsed_doc.metadata["type"],
                file_["createdTime"],
                file_["modifiedTime"],
                len(comments),
                len(open_comments),
            ]
            yield row


def update_sheet() -> None:
    """
    Get specs from Google Drive, parse the metadata on top of the document
    and write into a spreadsheet
    """
    drive = Drive()
    sheets = Sheets(spreadsheet_id=TRACKER_SPREADSHEET_ID)

    specs_sheet = sheets.get_sheet_by_title(SPECS_SHEET_TITLE)
    tmp_sheet = sheets.ensure_sheet_by_title(TMP_SHEET_TITLE)

    @tenacity.retry(
        stop=tenacity.stop_after_attempt(3),
        wait=tenacity.wait_incrementing(start=0.5, increment=0.8),
    )
    def _append_rows(rows):
        """Helper to retry extending the TMP_SHEET."""
        return sheets.insert_rows(rows, range=TMP_SHEET_TITLE)

    sheets.clear(sheet_id=tmp_sheet["properties"]["sheetId"])
    # Add headers
    _append_rows(
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
        ]
    )

    query_subfolders = (
        f"mimeType = 'application/vnd.google-apps.folder' "
        f"and '{TEAMS_FOLDER_ID}' in parents"
    )
    folders = drive.get_files(query=query_subfolders, fields=("id", "name"))

    existing_specs = {}

    spec_rows = specs_sheet.get("data", [{}])[0].get("rowData", [])

    # create a dict with the existing specs in the sheet, ignore the header
    for row in spec_rows[1:]:
        try:
            file_id = row["values"][2]["formattedValue"]
            existing_specs[file_id] = row
        except (KeyError, IndexError) as e:
            print("Error parsing row", e)

    # Insert rows in batches of 25, which is a magic number with no science
    # behind it.
    for rows in batched(
        _generate_spec_rows_for_folders(drive, folders, existing_specs),
        25,
    ):
        _append_rows(rows=rows)

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
