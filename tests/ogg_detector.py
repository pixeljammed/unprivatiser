#!/usr/bin/env python3
"""
Script to search for binary files without extensions and identify if they are OGG files.
"""

import os
import sys
from pathlib import Path

def is_ogg_file(file_path):
    """
    Check if a file is an OGG file by examining its magic bytes.
    OGG files start with 'OggS' (4F 67 67 53 in hex).
    """
    try:
        with open(file_path, 'rb') as f:
            # Read the first 4 bytes
            header = f.read(4)
            return header == b'OggS'
    except (IOError, OSError):
        return False

def has_no_extension(file_path):
    """
    Check if a file has no extension.
    """
    return '.' not in Path(file_path).name

def is_binary_file(file_path):
    """
    Check if a file is binary by looking for null bytes in the first 1024 bytes.
    """
    try:
        with open(file_path, 'rb') as f:
            chunk = f.read(1024)
            return b'\x00' in chunk
    except (IOError, OSError):
        return False

def search_ogg_files(root_folder):
    """
    Search for OGG files without extensions in the given folder and subfolders.
    """
    ogg_files = []
    root_path = Path(root_folder)
    
    if not root_path.exists():
        print(f"Error: Folder '{root_folder}' does not exist.")
        return ogg_files
    
    if not root_path.is_dir():
        print(f"Error: '{root_folder}' is not a directory.")
        return ogg_files
    
    print(f"Searching in: {root_path.absolute()}")
    print("Looking for binary files without extensions that are OGG files...")
    print("-" * 60)
    
    # Walk through all files in the directory tree
    for file_path in root_path.rglob('*'):
        if file_path.is_file():
            # Check if file has no extension
            if has_no_extension(file_path):
                # Check if it's a binary file
                if is_binary_file(file_path):
                    # Check if it's an OGG file
                    if is_ogg_file(file_path):
                        ogg_files.append(file_path)
                        print(f"Found OGG file: {file_path}")
    
    return ogg_files

def main():
    """
    Main function to handle command line arguments and execute the search.
    """
    if len(sys.argv) != 2:
        print("Usage: python ogg_detector.py <folder_path>")
        print("Example: python ogg_detector.py /path/to/search")
        sys.exit(1)
    
    folder_path = sys.argv[1]
    
    # Search for OGG files
    found_files = search_ogg_files(folder_path)
    
    # Summary
    print("-" * 60)
    print(f"Search completed. Found {len(found_files)} OGG file(s) without extensions.")
    
    if found_files:
        print("\nSummary of found files:")
        for i, file_path in enumerate(found_files, 1):
            file_size = file_path.stat().st_size
            print(f"{i}. {file_path} ({file_size:,} bytes)")
        
        # Optional: Ask if user wants to rename files
        print("\nWould you like to add .ogg extensions to these files? (y/n): ", end="")
        try:
            response = input().strip().lower()
            if response == 'y':
                rename_files(found_files)
        except KeyboardInterrupt:
            print("\nOperation cancelled.")

def rename_files(file_list):
    """
    Rename files by adding .ogg extension.
    """
    print("\nRenaming files...")
    success_count = 0
    
    for file_path in file_list:
        try:
            new_path = file_path.with_suffix('.ogg')
            if new_path.exists():
                print(f"Warning: {new_path} already exists, skipping {file_path}")
                continue
            
            file_path.rename(new_path)
            print(f"Renamed: {file_path} -> {new_path}")
            success_count += 1
        except OSError as e:
            print(f"Error renaming {file_path}: {e}")
    
    print(f"\nSuccessfully renamed {success_count} file(s).")

if __name__ == "__main__":
    main()
