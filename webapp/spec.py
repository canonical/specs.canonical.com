import logging

from bs4 import BeautifulSoup
from dateutil.parser import parse
from flask import abort

from webapp.google import Drive

logger = logging.getLogger(__name__)

specs_status = (
    "active",
    "approved",
    "braindump",
    "completed",
    "drafting",
    "obsolete",
    "pending review",
    "rejected",
)

spec_types = (
    "implementation",
    "product requirement",
    "standard",
    "informational",
    "process",
)


class Spec:
    def __init__(self, google_drive: Drive, document_id: str):
        self.document_id = document_id
        self.url = f"https://docs.google.com/document/d/{document_id}"
        self.metadata = {
            "index": "",
            "title": "",
            "status": "",
            "authors": [],
            "type": "",
            "created": "",
        }

        try:
            raw_html = google_drive.doc_html(document_id)
        except Exception as e:
            err = "Error. Document doesn't exist."
            logger.error(err, exc_info=e)
            abort(404, description=err)
        self.html = BeautifulSoup(raw_html, features="lxml")
        self.clean()
        self.parse_metadata()

    def clean(self):
        def empty_tags_selector(tag):
            return (  # noqa
                not tag.contents or len(tag.get_text(strip=True)) <= 0
            ) and tag.name not in ["br", "img", "hr"]

        for element in self.html.findAll(empty_tags_selector):
            element.decompose()

    def parse_metadata(self):
        table = self.html.select_one("table")
        if not table:
            raise ValueError("No metadata table found in document")
        for table_row in table.select("tr"):
            cells = table_row.select("td")
            # invalid format | name | value |, ignoring the row
            if len(cells) != 2:
                continue
            attr_name, attr_value = cells
            attr_name = attr_name.text.lower().strip()

            if attr_name == "authors":
                # Select all span elements
                authors_span_list = [
                    value.select("span") for value in attr_value
                ]
                authors_list = []
                for author in authors_span_list[0]:
                    authors_list.append(author.text.strip())
                # Remove empty items (i.e ",") from list
                authors_list = [x for x in authors_list if x != ","]
                attr_value = ",".join(authors_list)
            else:
                attr_value = attr_value.text.strip()

            if attr_name not in self.metadata:
                continue

            if attr_name in ["index", "title"]:
                self.metadata[attr_name] = attr_value
            elif attr_name == "status":
                if attr_value.lower() in specs_status:
                    self.metadata["status"] = attr_value
                else:
                    self.metadata["status"] = "unknown"
                    self.metadata["statusMessage"] = attr_value
            elif attr_name == "authors":
                self.metadata["authors"] = [
                    author.strip() for author in attr_value.split(",")
                ]
            elif attr_name == "type":
                if attr_value.lower() in spec_types:
                    self.metadata["type"] = attr_value
                else:
                    self.metadata["type"] = "unknown"
            elif attr_name == "created":
                try:
                    self.metadata["created"] = parse(attr_value, fuzzy=True)
                except Exception as e:
                    logger.error("Unable to parse date", exc_info=e)
                    self.metadata["created"] = "unknown"

        table.decompose()
