import subprocess
root_dir = "/home/cdsw/monitoring-studio" if os.getenv("IS_COMPOSABLE", "") != "" else "/home/cdsw"

print("Starting App")
print(subprocess.run([f"bash {root_dir}/scripts/start_app.sh"], shell=True, check=True))
