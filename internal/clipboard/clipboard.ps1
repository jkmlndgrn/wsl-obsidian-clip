Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

[Console]::Out.WriteLine("READY")
[Console]::Out.Flush()

while ($true) {
    $readTask = [Console]::In.ReadLineAsync()
    while (-not $readTask.IsCompleted) {
        [System.Windows.Forms.Application]::DoEvents()
        Start-Sleep -Milliseconds 50
    }
    $line = $readTask.Result
    if ($line -eq $null -or $line -eq "EXIT") {
        break
    }

    if ($line -eq "CHECK") {
        # Check for image in clipboard
        if (-not [System.Windows.Forms.Clipboard]::ContainsImage()) {
            [Console]::Out.WriteLine("NONE")
            [Console]::Out.Flush()
            continue
        }

        # Skip spreadsheet app clipboard captures
        $dataObj = [System.Windows.Forms.Clipboard]::GetDataObject()
        if ($dataObj -ne $null) {
            $formats = $dataObj.GetFormats()
            if ($formats -contains "XML Spreadsheet" -or
                $formats -contains "Csv" -or
                ($formats -contains "HTML Format" -and [System.Windows.Forms.Clipboard]::ContainsText())) {
                [Console]::Out.WriteLine("NONE")
                [Console]::Out.Flush()
                continue
            }
        }

        # Fingerprint: skip if our previous write is still there
        # (image + text + file drop = our enriched clipboard from a previous cycle)
        if ([System.Windows.Forms.Clipboard]::ContainsText() -and
            [System.Windows.Forms.Clipboard]::ContainsFileDropList()) {
            [Console]::Out.WriteLine("NONE")
            [Console]::Out.Flush()
            continue
        }

        # Get image and convert to PNG base64
        $img = [System.Windows.Forms.Clipboard]::GetImage()
        if ($img -eq $null) {
            [Console]::Out.WriteLine("NONE")
            [Console]::Out.Flush()
            continue
        }
        try {
            $ms = New-Object System.IO.MemoryStream
            $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
            $bytes = $ms.ToArray()
            $b64 = [Convert]::ToBase64String($bytes)
            [Console]::Out.WriteLine("IMAGE")
            [Console]::Out.WriteLine($b64)
            [Console]::Out.WriteLine("END")
            [Console]::Out.Flush()
        } finally {
            $ms.Dispose()
            $img.Dispose()
        }
    }
}
