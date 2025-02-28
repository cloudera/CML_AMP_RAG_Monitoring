import subprocess
import os

root_dir = "/home/cdsw/monitoring-studio" if os.getenv("IS_COMPOSABLE", "") != "" else "/home/cdsw"
os.makedirs(root_dir, exist_ok=True)
os.chdir(root_dir)

print(
    subprocess.run(
        [f"bash {root_dir}/scripts/install_base.sh"], shell=True, check=True
    )
)
print("Installing base dependencies complete")
