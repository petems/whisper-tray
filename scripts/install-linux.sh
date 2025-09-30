#!/bin/bash
# Linux installation script

set -e

echo "Installing WhisperTray..."

# Detect package manager
if command -v apt-get &> /dev/null; then
    sudo apt-get update
    sudo apt-get install -y portaudio19-dev libx11-dev libxtst-dev
elif command -v dnf &> /dev/null; then
    sudo dnf install -y portaudio-devel libX11-devel libXtst-devel
elif command -v pacman &> /dev/null; then
    sudo pacman -S --noconfirm portaudio libx11 libxtst
fi

# Copy binary
sudo cp whisper-tray /usr/local/bin/
sudo chmod +x /usr/local/bin/whisper-tray

# Create desktop entry
mkdir -p ~/.local/share/applications
cat > ~/.local/share/applications/whisper-tray.desktop <<EOF
[Desktop Entry]
Type=Application
Name=WhisperTray
Comment=Local voice dictation
Exec=/usr/local/bin/whisper-tray
Icon=whisper-tray
Terminal=false
Categories=Utility;
EOF

echo "✓ Installation complete!"
echo "Run 'whisper-tray' to start"