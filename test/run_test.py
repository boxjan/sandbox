#!/usr/bin/python3

from normal.normal import Normal
from syscall.syscall import Syscall

def normal_test():
    test = Normal()
    test.main()

if __name__ == "__main__":
    normal_test()
    print ("All is OK")