import os


PRIVATE_KEY_ID = os.getenv("PRIVATE_KEY_ID")
PRIVATE_KEY = os.getenv("PRIVATE_KEY")
SERVICE_ACCOUNT_INFO = {
    "type": "service_account",
    "project_id": "roadmap-270011",
    "private_key_id": PRIVATE_KEY_ID,
    "private_key": PRIVATE_KEY,
    "client_email": "specs-reader@roadmap-270011.iam.gserviceaccount.com",
    "client_id": "112404606310881291739",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": (
        "https://www.googleapis.com/oauth2/v1/certs"
    ),
    "client_x509_cert_url": (
        "https://www.googleapis.com/robot/v1/metadata"
        "/x509/specs-reader%40roadmap-270011.iam.gserviceaccount.com"
    ),
}
