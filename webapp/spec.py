import io

from apiclient.http import MediaIoBaseDownload
from bs4 import BeautifulSoup
from google.oauth2 import service_account
from googleapiclient.discovery import build

from dateutil.parser import parse

from flask import abort

from webapp.settings import SERVICE_ACCOUNT_INFO


class GoogleDrive:
    def __init__(self):
        scopes = [
            "https://www.googleapis.com/auth/drive.readonly",
        ]
        credentials = service_account.Credentials.from_service_account_info(
            SERVICE_ACCOUNT_INFO, scopes=scopes
        )
        self.service = build(
            "drive", "v2", credentials=credentials, cache_discovery=False
        )

    def doc_html(self, document_id):
        request = self.service.files().export_media(
            fileId=document_id, mimeType="text/html"
        )
        fh = io.BytesIO()
        downloader = MediaIoBaseDownload(fh, request)
        done = False
        while done is False:
            _, done = downloader.next_chunk()
        html = fh.getvalue().decode("utf-8")
        comments = (
            self.service.comments()
            .list(fileId=document_id)
            .execute()
            .get("items", [])
        )
        return html, comments


class Spec:
    html: BeautifulSoup
    metadata = {
        "index": "",
        "title": "",
        "status": "",
        "authors": [],
        "type": "",
        "created": "",
    }
    url = "https://docs.google.com/document/d/"

    def __init__(self, google_drive: GoogleDrive, document_id: str):
        try:
            raw_html, _ = google_drive.doc_html(document_id)
        except Exception as e:
            err = "Error. Document doesn't exist."
            print(f"{err}\n {e}")
            abort(404, description=err)
        self.url = f"{self.url}/{document_id}"
        self.html = BeautifulSoup(raw_html, features="lxml")
        self.clean()
        self.parse_metadata()

    def clean(self):
        empty_tags_selector = lambda tag: (
            not tag.contents or len(tag.get_text(strip=True)) <= 0
        ) and tag.name not in ["br", "img", "hr"]
        for element in self.html.findAll(empty_tags_selector):
            element.decompose()

    def parse_metadata(self):
        table = self.html.select_one("table")
        for table_row in table.select("tr"):
            cells = table_row.select("td")
            # invalid format | name | value |, ignoring the row
            if len(cells) != 2:
                continue
            attr_name, attr_value = cells
            attr_name = attr_name.text.lower().strip()
            attr_value = attr_value.text.strip()
            attr_value_lower_case = attr_value.lower()
            if attr_name in self.metadata:
                if attr_name in ["index", "title"]:
                    self.metadata[attr_name] = attr_value
                elif attr_name == "status":
                    if attr_value_lower_case in [
                        "approved",
                        "pending review",
                        "drafting",
                        "braindump",
                        "unknown",
                    ]:
                        self.metadata["status"] = attr_value_lower_case
                    else:
                        self.metadata["status"] = "unknown"
                elif attr_name == "authors":
                    self.metadata["authors"] = [
                        author.strip() for author in attr_value.split(",")
                    ]
                elif attr_name == "type":
                    if attr_value_lower_case in [
                        "standard",
                        "informational",
                        "process",
                    ]:
                        self.metadata["type"] = attr_value_lower_case
                    else:
                        self.metadata["type"] = "unknown"
                elif attr_name == "created":
                    self.metadata["created"] = parse(
                        attr_value_lower_case, fuzzy=True
                    )
        table.decompose()
