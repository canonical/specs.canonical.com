import hashlib
import io
import os
import tempfile
from typing import List

from apiclient.http import MediaIoBaseDownload
from google.oauth2 import service_account
from googleapiclient.discovery import build

from webapp.settings import SERVICE_ACCOUNT_INFO


class Drive:
    def __init__(self):
        scopes = [
            "https://www.googleapis.com/auth/drive.readonly",
        ]
        credentials = service_account.Credentials.from_service_account_info(
            SERVICE_ACCOUNT_INFO, scopes=scopes
        )
        self.service = build(
            "drive", "v3", credentials=credentials, cache_discovery=False
        )

    def get_comments(self, file_id, fields=None):
        fields = f"comments({','.join(fields)})" if fields else None

        response = (
            self.service.comments()
            .list(fileId=file_id, fields=fields)
            .execute()
        )

        return response.get("comments", [])

    def doc_html(self, document_id):
        request = self.service.files().export(
            fileId=document_id, mimeType="text/html"
        )
        fh = io.BytesIO()
        downloader = MediaIoBaseDownload(fh, request)
        done = False
        while done is False:
            _, done = downloader.next_chunk()
        html = fh.getvalue().decode("utf-8")

        return html

    def get_files(self, query, fields=None):
        fields = f"files({','.join(fields)})" if fields else None
        fields = f"nextPageToken, {fields}" if fields else "nextPageToken"

        page_token = None
        files = []
        while True:
            response = (
                self.service.files()
                .list(
                    supportsAllDrives=True,
                    includeItemsFromAllDrives=True,
                    fields=fields,
                    pageToken=page_token,
                    q=query,
                )
                .execute()
            )
            files.extend(response.get("files", []))
            page_token = response.get("nextPageToken", None)

            if page_token is None:
                break

        return files


class DiscoveryCache:
    """
    Unix file-based cache for use with the API Discovery service
    See https://github.com/googleapis/google-api-python-client/issues/325#issuecomment-419387788
    """  # noqa

    def filename(self, url):
        return os.path.join(
            tempfile.gettempdir(),
            "google_api_discovery_" + hashlib.md5(url.encode()).hexdigest(),
        )

    def get(self, url):
        try:
            with open(self.filename(url), "rb") as f:
                return f.read().decode()
        except FileNotFoundError:
            return None

    def set(self, url, content):
        with tempfile.NamedTemporaryFile(delete=False) as f:
            f.write(content.encode())
            f.flush()
            os.fsync(f)
        os.rename(f.name, self.filename(url))


class Sheets:
    def __init__(self, spreadsheet_id):
        self.spreadsheet_id = spreadsheet_id

        scopes = ["https://www.googleapis.com/auth/spreadsheets"]
        creds = service_account.Credentials.from_service_account_info(
            SERVICE_ACCOUNT_INFO, scopes=scopes
        )
        service = build(
            "sheets", "v4", credentials=creds, cache=DiscoveryCache()
        )
        self.spreadsheets = service.spreadsheets()

    def _batch_update(self, body):
        response = self.spreadsheets.batchUpdate(
            spreadsheetId=self.spreadsheet_id, body=body
        ).execute()
        return response

    def ensure_sheet_by_title(self, title, *args, **kwargs) -> dict:
        """
        Returns an existing sheet matching the title if it can be found, else
        create it.
        """
        try:
            return self.get_sheet_by_title(title, *args, **kwargs)
        except StopIteration:
            # no sheet found with that name
            self.create_sheet(title)
            return self.get_sheet_by_title(title, *args, **kwargs)

    def get_sheet_by_title(self, title, ranges=None) -> dict:
        """
        Return sheet with a given title
        """
        spreadsheet = self.spreadsheets.get(
            spreadsheetId=self.spreadsheet_id,
            ranges=ranges,
            includeGridData=True,
        ).execute()

        return next(
            s
            for s in spreadsheet["sheets"]
            if s["properties"]["title"] == title
        )

    def delete_sheets(self, sheet_ids_to_delete: list):
        sheets = []
        for sheet_id in sheet_ids_to_delete:
            sheets.append({"deleteSheet": {"sheetId": sheet_id}})

        delete_sheet_requests = {"requests": sheets}
        self._batch_update(body=delete_sheet_requests)

    def create_sheet(self, sheet_title: str) -> str:
        add_sheet_request = {
            "requests": [{"addSheet": {"properties": {"title": sheet_title}}}]
        }

        response = self._batch_update(add_sheet_request)

        new_sheet = response["replies"][0]["addSheet"]
        return new_sheet

    def clear(self, sheet_id: str) -> None:
        """
        Delete all content in a sheet
        """
        body = {
            "requests": [
                {
                    "updateCells": {
                        "range": {"sheetId": sheet_id},
                        "fields": "userEnteredValue",
                    }
                }
            ]
        }

        self._batch_update(body)

    def insert_rows(self, rows: List[List[str]], range: str) -> None:
        """
        Append rows to the end of the sheet
        """
        self.spreadsheets.values().append(
            spreadsheetId=self.spreadsheet_id,
            body={"values": rows},
            range=range,
            valueInputOption="RAW",
        ).execute()

    def update_sheet_name(self, sheet_id: str, new_name: str) -> None:
        """
        Change name of a sheet
        """
        body = {
            "requests": [
                {
                    "updateSheetProperties": {
                        "properties": {
                            "sheetId": sheet_id,
                            "title": new_name,
                        },
                        "fields": "title",
                    }
                }
            ]
        }

        self._batch_update(body)
