import json
from datetime import datetime

from webapp.google import Sheets
from webapp.settings import TRACKER_SPREADSHEET_ID, SPECS_SHEET_TITLE


def get_value_row(row, type):
    if row:
        if type == datetime:
            if "formattedValue" in row:
                return datetime.strptime(
                    row["formattedValue"], "%Y-%m-%dT%H:%M:%S.%fZ"
                ).strftime("%d %b %Y %H:%M")
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

def generate_specs(sheet):
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


if __name__ == "__main__":
    spreadsheet = Sheets(spreadsheet_id=TRACKER_SPREADSHEET_ID) 
    
    RANGE = "A2:M" 
    sheet = spreadsheet.get_sheet_by_title(
        title=SPECS_SHEET_TITLE, ranges=[f"{SPECS_SHEET_TITLE}!{RANGE}"]
    )
    
    specs = list(generate_specs(sheet))

    with open("specs.json", "w") as f:
        json.dump(specs, f, indent=4)

