import subprocess
import cmlapi
import os

print(subprocess.run(["bash", "/home/cdsw/scripts/refresh_project.sh"], check=True))

print(
    "Project refresh complete. Restarting the Monitoring Application to pick up changes, if this isn't the initial deployment.")

client = cmlapi.default_client()
project_id = os.environ['CDSW_PROJECT_ID']
apps = client.list_applications(project_id=project_id)
if len(apps.applications) > 0:
    # todo: handle case where there are multiple apps
    app_id = apps.applications[0].id
    print("Restarting app with ID: ", app_id)
    client.restart_application(application_id=app_id, project_id=project_id)
else:
    print("No applications found to restart. This is likely the initial deployment.")

