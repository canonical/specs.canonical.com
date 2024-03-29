import copy
import json
import os
import flask

from flask import render_template, jsonify, abort, redirect
from canonicalwebteam.flask_base.app import FlaskBase

from cachetools import cached, TTLCache

from webapp.authors import parse_authors, unify_authors
from webapp.spec import Spec
from webapp.sso import init_sso
from webapp.update import update_sheet
from webapp.google import Drive

CACHE_TTL = 60 * 30

drive = Drive()

app = FlaskBase(
    __name__,
    "webteam.canonical.com",
    template_folder="../templates",
    static_folder="../static",
)

init_sso(app)

SPECS_FILE = "specs.json"
all_specs = []
if os.path.exists(SPECS_FILE):
    with open(SPECS_FILE) as f:
        all_specs = json.load(f)


@app.route("/")
def index():
    specs = []
    teams = set()
    for spec in copy.deepcopy(all_specs):
        spec["authors"] = parse_authors(spec["authors"])
        if spec["folderName"]:
            teams.add(spec["folderName"])
        specs.append(spec)
    specs = unify_authors(specs)
    teams = sorted(teams)

    return render_template("index.html", specs=specs, teams=teams)


@app.route("/spec/<spec_name>")
def spec(spec_name):
    for spec in all_specs:
        if spec_name.upper() == spec["index"]:
            return redirect(spec["fileURL"])
    else:
        abort(404)


@app.route("/spec-details/<document_id>")
# Cache for 30 minutes
@cached(cache=TTLCache(maxsize=128, ttl=CACHE_TTL))
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


@app.route("/my-specs")
def my_specs():
    specs = []
    teams = set()
    user = flask.session["openid"]
    for spec in copy.deepcopy(all_specs):
        spec["authors"] = parse_authors(spec["authors"])
        if user["fullname"] in spec["authors"]:
            if spec["folderName"]:
                teams.add(spec["folderName"])
            specs.append(spec)
    specs = unify_authors(specs)
    teams = sorted(teams)

    return render_template("index.html", specs=specs, teams=teams)


@app.cli.command("update-spreadsheet")
def update_spreadsheet():
    """
    Update the spreadsheet that contains the specs information
    """
    update_sheet()
