from flask import abort
from dateutil.parser import parse
from bs4 import BeautifulSoup


from webapp.google import Drive


specs_status = (
    "braindump",
    "drafting",
    "pending review",
    "approved",
    "rejected",
    "completed",
    "obsolete",
)

spec_types = (
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
            print(f"{err}\n {e}")
            abort(404, description=err)
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

            if attr_name in self.metadata:
                if attr_name in ["index", "title"]:
                    self.metadata[attr_name] = attr_value
                elif attr_name == "status":
                    if attr_value.lower() in specs_status:
                        self.metadata["status"] = attr_value
                    else:
                        self.metadata["status"] = "unknown"
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
                        self.metadata["created"] = parse(
                            attr_value, fuzzy=True
                        )
                    except Exception as e:
                        print(e)
                        self.metadata["created"] = "unknown"

        table.decompose()
