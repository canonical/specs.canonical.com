import os
from datetime import datetime

from flask import render_template, jsonify, abort, redirect
from canonicalwebteam.flask_base.app import FlaskBase

from webapp.authors import parse_authors, unify_authors
from webapp.spec import Spec
from webapp.sso import init_sso
from webapp.update import update_sheet
from webapp.google import Drive, Sheets
from webapp.contants import TRACKER_SPREADSHEET_ID, SPECS_SHEET_TITLE

DEPLOYMENT_ID = os.getenv(
    "DEPLOYMENT_ID",
    "AKfycbw5ph73HX2plnYE1Q03K7M8BQhlrp12Dck27bukPWCbXzBdRgP1N456fPiipR9J2H7q",
)
SPECS_API = f"https://script.google.com/macros/s/{DEPLOYMENT_ID}/exec"


spreadsheet = Sheets(spreadsheet_id=TRACKER_SPREADSHEET_ID)
drive = Drive()

app = FlaskBase(
    __name__,
    "webteam.canonical.com",
    template_folder="../templates",
    static_folder="../static",
)

init_sso(app)


def get_value_row(row, type):
    if row:
        if type == datetime:
            if "formattedValue" in row:
                return datetime.strptime(
                    row["formattedValue"], "%Y-%m-%dT%H:%M:%S.%fZ"
                ).strftime("%d %b %Y")
        elif "userEnteredValue" in row:
            if "stringValue" in row["userEnteredValue"]:
                return type(row["userEnteredValue"]["stringValue"])
            if "numberValue" in row["userEnteredValue"]:
                return type(row["userEnteredValue"]["numberValue"])

    return ""


def index_in_list(a_list, index):
    return index < len(a_list)


def is_spec(row):
    """Check that file name exists."""

    return "userEnteredValue" in row[1]


def _generate_specs():
    RANGE = "A2:M1000"
    COLUMNS = [
        ("folderName", str),
        ("fileName", str),
        ("fileID", str),
        ("fileURL", str),
        ("index", str),
        ("title", str),
        ("status", str),
        ("authors", str),
        ("type", str),
        ("created", datetime),
        ("lastUpdated", datetime),
        ("numberOfComments", int),
        ("openComments", int),
    ]

    sheet = spreadsheet.get_sheet_by_title(
        title=SPECS_SHEET_TITLE, ranges=[f"{SPECS_SHEET_TITLE}!{RANGE}"]
    )

    for row in sheet["data"][0]["rowData"]:
        if "values" in row and is_spec(row["values"]):
            spec = {}
            for column_index in range(len(COLUMNS)):
                (column, type) = COLUMNS[column_index]
                spec[column] = get_value_row(
                    row["values"][column_index]
                    if index_in_list(row["values"], column_index)
                    else None,
                    type,
                )
            yield spec


@app.route("/")
def index():
    specs = []
    teams = set()
    for spec in _generate_specs():
        spec["authors"] = parse_authors(spec["authors"])
        if spec["folderName"]:
            teams.add(spec["folderName"])
        specs.append(spec)
    specs = unify_authors(specs)
    teams = sorted(teams)

    return render_template("index.html", specs=specs, teams=teams)


@app.route("/spec/<spec_name>")
def spec(spec_name):
    for spec in _generate_specs():
        if spec_name == spec["index"]:
            return redirect(spec["fileURL"])
    else:
        abort(404)


@app.route("/spec-details/<document_id>")
def get_document(document_id):
    try:
        spec = Spec(drive, document_id)
    except Exception as e:
        err = "Error fetching document, try again."
        print(f"{err}\n {e}")
        abort(500, description=err)
    payload = {
        "metadata": spec.metadata,
        "url": spec.url,
        "html": spec.html.encode("utf-8").decode(),
    }
    return jsonify(payload)


@app.cli.command("update-spreadsheet")
def update_spreadsheet():
    """
    Update the spreadsheet that contains the specs information
    """
    update_sheet()
