#!/usr/bin/env python3
"""Reduce each capture to a 256-colour palette in place.

A 2880x1800 RGB capture of the UI runs past the 512KB source budget; the same
picture on a palette is a third of the size and indistinguishable at any
density the site serves. usage: quantize.py <png>...
"""
import os
import sys

from PIL import Image

for path in sys.argv[1:]:
    image = Image.open(path).convert("RGB")
    palette = image.quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
    palette.save(path, optimize=True)
    print(f"{path}: {os.path.getsize(path) // 1024} KB")
