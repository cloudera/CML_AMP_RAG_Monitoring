import subprocess
import os

root_dir = "/home/cdsw/ml-monitoring" if os.getenv("IS_COMPOSABLE", "") != "" else "/home/cdsw"
os.chdir(root_dir)

print(
    subprocess.run(
        ["bash /home/cdsw/scripts/install_base.sh"], shell=True, check=True
    )
)
print("Installing base dependencies complete")
