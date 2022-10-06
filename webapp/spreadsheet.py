import os
import hashlib
import tempfile
from googleapiclient.discovery import build
from google.oauth2 import service_account

from webapp.settings import SERVICE_ACCOUNT_INFO


SCOPES = ["https://www.googleapis.com/auth/spreadsheets.readonly"]


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


def get_sheet():
    service_account_info = SERVICE_ACCOUNT_INFO
    creds = service_account.Credentials.from_service_account_info(
        service_account_info, scopes=SCOPES
    )
    service = build("sheets", "v4", credentials=creds, cache=DiscoveryCache())

    sheet = service.spreadsheets()

    return sheet
