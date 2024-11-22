import subprocess

print(
    subprocess.run(
        ["bash /home/cdsw/scripts/install_base.sh"], shell=True, check=True
    )
)
print("Installing base dependencies complete")
