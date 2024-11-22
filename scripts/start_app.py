import subprocess

print("Starting App")
print(subprocess.run(["bash /home/cdsw/scripts/start_app.sh"], shell=True, check=True))
