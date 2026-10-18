#!/usr/bin/env python3
"""Draws the app icon: a small constellation on a night sky, one star lit,
as the skill map draws an unlock. A placeholder until there is a designed
icon; it exists because App Store Connect refuses a build without one.

Needs Pillow (pip install pillow). Writes a 1024 px opaque PNG:

  ios/scripts/make-icon.py ios/Hefesto/Assets.xcassets/AppIcon.appiconset/AppIcon.png
"""

import math
import sys

from PIL import Image, ImageDraw, ImageFilter

SIZE = 1024
SS = 4  # supersampling for smooth edges
N = SIZE * SS

TOP, BOTTOM = (8, 10, 28), (26, 30, 62)
LINE = (255, 214, 10)
STAR = (255, 214, 10)
DIM = (255, 255, 255)

# The constellation, in icon coordinates: three lit steps and two still dim.
STARS = [(250, 760, "lit"), (420, 600, "lit"), (560, 420, "lit"), (760, 300, "dim"), (640, 700, "dim")]
LINKS = [(0, 1, "lit"), (1, 2, "lit"), (2, 3, "dim"), (1, 4, "dim")]
BRIGHT = 2  # the newest unlock


def s(v):
    return v * SS


def main(out):
    img = Image.new("RGB", (N, N))
    px = ImageDraw.Draw(img)
    for y in range(N):
        t = y / (N - 1)
        px.line([(0, y), (N, y)], fill=tuple(round(a + (b - a) * t) for a, b in zip(TOP, BOTTOM)))

    # A faint fixed star field.
    field = Image.new("RGBA", (N, N), (0, 0, 0, 0))
    fd = ImageDraw.Draw(field)
    state = 7
    for _ in range(90):
        state = (state * 6364136223846793005 + 1442695040888963407) % 2**64
        x = (state >> 11) / 2**53 * SIZE
        state = (state * 6364136223846793005 + 1442695040888963407) % 2**64
        y = (state >> 11) / 2**53 * SIZE
        r = 1.5 + ((state >> 20) % 100) / 100 * 2.5
        fd.ellipse([s(x - r), s(y - r), s(x + r), s(y + r)], fill=DIM + (70,))
    img.paste(field, (0, 0), field)

    # Glow under the lit lines and the bright star.
    glow = Image.new("RGBA", (N, N), (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    for a, b, kind in LINKS:
        if kind == "lit":
            (x1, y1, _), (x2, y2, _) = STARS[a], STARS[b]
            gd.line([s(x1), s(y1), s(x2), s(y2)], fill=LINE + (150,), width=s(36))
    bx, by, _ = STARS[BRIGHT]
    gd.ellipse([s(bx - 150), s(by - 150), s(bx + 150), s(by + 150)], fill=STAR + (140,))
    glow = glow.filter(ImageFilter.GaussianBlur(s(40)))
    img.paste(glow, (0, 0), glow)

    d = ImageDraw.Draw(img, "RGBA")
    for a, b, kind in LINKS:
        (x1, y1, _), (x2, y2, _) = STARS[a], STARS[b]
        if kind == "lit":
            d.line([s(x1), s(y1), s(x2), s(y2)], fill=LINE + (255,), width=s(12))
        else:
            # Dashed, like a path not yet walked.
            length = math.hypot(x2 - x1, y2 - y1)
            steps = int(length // 34)
            for i in range(steps):
                t0, t1 = i / steps, (i + 0.5) / steps
                d.line([s(x1 + (x2 - x1) * t0), s(y1 + (y2 - y1) * t0),
                        s(x1 + (x2 - x1) * t1), s(y1 + (y2 - y1) * t1)], fill=DIM + (110,), width=s(8))
    for i, (x, y, kind) in enumerate(STARS):
        if i == BRIGHT:
            continue
        r = 34 if kind == "lit" else 26
        if kind == "lit":
            d.ellipse([s(x - r), s(y - r), s(x + r), s(y + r)], fill=STAR + (255,))
        else:
            d.ellipse([s(x - r), s(y - r), s(x + r), s(y + r)], fill=DIM + (40,), outline=DIM + (150,), width=s(6))

    # The newest unlock: a four-pointed star.
    arm, waist = 150, 34
    pts = []
    for k in range(8):
        ang = math.pi / 4 * k - math.pi / 2
        r = arm if k % 2 == 0 else waist
        pts.append((s(bx + r * math.cos(ang)), s(by + r * math.sin(ang))))
    d.polygon(pts, fill=(255, 244, 200, 255))
    d.ellipse([s(bx - 22), s(by - 22), s(bx + 22), s(by + 22)], fill=(255, 255, 255, 255))

    img.resize((SIZE, SIZE), Image.LANCZOS).save(out, "PNG", optimize=True)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "AppIcon.png")
