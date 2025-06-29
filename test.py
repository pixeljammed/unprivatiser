import os, shutil

SOURCE_DIR = r"c:\Users\pixel\AppData\Local\Temp\Roblox\sounds"
NEW_DIR = r"c:\Users\pixel\OneDrive\Documents\sounds_lol"


files = os.listdir(SOURCE_DIR)
while True:
    new_files = os.listdir(SOURCE_DIR)
    if files != new_files:
        print("DIFF!!")