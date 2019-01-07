#!/usr/bin/python3

import os

pid = os.fork()

if pid < 0:
    print("Fork Error!")
    exit(1)

if pid > 0:
    print("Father")
else:
    print("Son")