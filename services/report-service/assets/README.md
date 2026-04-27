# Assets Directory

This directory contains static assets used by the report-service for report generation.

## Logo

The `logo.png` file is used as the company logo in generated PDF reports.

### Requirements

- **Format**: PNG (with transparent background recommended)
- **Size**: 200x60 pixels (will be scaled to 35mm width in PDF)
- **Background**: Transparent or white
- **Resolution**: 300 DPI recommended for print quality

### Placeholder

If you don't have a logo yet, you can create a simple placeholder:

```bash
# Using ImageMagick to create a simple text-based logo placeholder
convert -size 200x60 xc:white \
  -font Helvetica -pointsize 24 -fill "#003366" \
  -gravity center \
  -annotate 0 "Company Logo" \
  assets/logo.png
```

Or use any image editing tool to create your logo and save it as `logo.png` in this directory.

## Other Assets

Additional assets (watermarks, backgrounds, etc.) can be added here and referenced in the template files.
