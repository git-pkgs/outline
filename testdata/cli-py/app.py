import subprocess

from pkg import loader


def handler(name):
    rec = loader.load(name)
    subprocess.run(rec)
