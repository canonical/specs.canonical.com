import datetime
import logging
from typing import Dict, List

import jsonlines
import tenacity

from webapp.build_specs import save_specs_locally
from webapp.google import Drive, Sheets
from webapp.settings import (
    SPECS_FILE,
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


logger = logging.getLogger(__name__)


def _generate_spec_rows_for_folders(
    drive: Drive,
    folders: List[Dict],
    existing_specs_id_index: Dict[str, List],
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
        logger.info("Found %s documents in %s", len(files), folder["name"])

        for file_ in files:
            existing_spec = existing_specs_id_index.get(file_["id"])
            file_changed_since_last_sync = (
                existing_spec is None
                or existing_spec["lastUpdated"] != file_["modifiedTime"]
            )

            # file has not been modified since last update
            # use the same row as before
            if existing_spec and not file_changed_since_last_sync:
                row = [
                    folder["name"],
                    file_["name"],
                    file_["id"],
                    file_["webViewLink"],
                    existing_spec["index"],
                    existing_spec["title"],
                    existing_spec["status"],
                    existing_spec["authors"],
                    existing_spec["type"],
                    file_["createdTime"],
                    file_["modifiedTime"],
                    existing_spec["numberOfComments"],
                    existing_spec["openComments"],
                ]
                logger.info("Using existing row for %s", file_["name"])
                yield row
                continue

            try:
                comments = drive.get_comments(
                    file_id=file_["id"], fields=("resolved",)
                )
                open_comments = [c for c in comments if not c["resolved"]]
                parsed_doc = Spec(google_drive=drive, document_id=file_["id"])
            except Exception as e:
                logger.error(
                    "Unable to parse document: %s", file_["name"], exc_info=e
                )
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
    start_time = datetime.datetime.now()
    logger.info("Update sheet started...")
    drive = Drive()
    sheets = Sheets(spreadsheet_id=TRACKER_SPREADSHEET_ID)

    specs_sheet = sheets.ensure_sheet_by_title(SPECS_SHEET_TITLE)
    tmp_sheet_id = create_tmp_sheet(sheets)

    save_specs_locally()

    @tenacity.retry(
        stop=tenacity.stop_after_attempt(3),
        wait=tenacity.wait_incrementing(start=0.5, increment=0.8),
    )
    def _append_rows(rows):
        """Helper to retry extending the TMP_SHEET."""
        return sheets.insert_rows(rows, range=TMP_SHEET_TITLE)

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
    logger.info("Finding subfolders of %s", TEAMS_FOLDER_ID)
    folders = drive.get_files(query=query_subfolders, fields=("id", "name"))
    logger.info("Found %s subfolders", len(folders))

    # create a dict with the existing specs in the sheet, ignore the header
    existing_specs = {}
    with jsonlines.open(SPECS_FILE) as reader:
        for spec in reader:
            existing_specs = existing_specs[spec["fileID"]] = spec

    logger.info(
        "Found %s existing specs in the current sheet",
        len(existing_specs.values()),
    )
    # Insert rows in batches of 25, which is a magic number with no science
    # behind it.
    total_specs = 0
    for rows in batched(
        _generate_spec_rows_for_folders(drive, folders, existing_specs),
        25,
    ):
        total_specs += len(rows)
        _append_rows(rows=rows)

    logger.info("Inserted %s specs in the sheet", total_specs)
    elapsed_time_seconds = (datetime.datetime.now() - start_time).seconds
    logger.info("Updated the sheet in %s seconds", elapsed_time_seconds)

    # delete the current main source sheet
    sheets.delete_sheets([specs_sheet["properties"]["sheetId"]])

    # rename the temporary sheet to the main source sheet
    sheets.update_sheet_name(
        sheet_id=tmp_sheet_id,
        new_name=SPECS_SHEET_TITLE,
    )


def create_tmp_sheet(sheets) -> str:
    """Function to create a temporary sheet.

    Validate if the `TMP_SHEET_TITLE` exists, if yes, delete.
    Create a new sheet with name `TMP_SHEET_TITLE`.

    Returns:
        ID of the `TMP_SHEET_TITLE` sheet
    """
    tmp_sheet = sheets.get_sheet_by_title(TMP_SHEET_TITLE)
    if tmp_sheet:
        sheets.delete_sheets([tmp_sheet["properties"]["sheetId"]])
    tmp_sheet = sheets.create_sheet(TMP_SHEET_TITLE)
    tmp_sheet_id = tmp_sheet["properties"]["sheetId"]
    return tmp_sheet_id
