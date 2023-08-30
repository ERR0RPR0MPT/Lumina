import base64
import sys

from PIL import Image
from pyzbar import pyzbar


if __name__ == "__main__":
    image = Image.open(sys.argv[1])
    gray_image = image.convert('L')
    width, height = gray_image.size
    updated_image = gray_image.resize((width * 4, height * 4))
    barcodes = pyzbar.decode(updated_image, symbols=[pyzbar.ZBarSymbol.QRCODE])
    if len(barcodes) > 1:
        print("检测到多个二维码，数据可能损坏")
    if len(barcodes) == 0:
        print("未检测到二维码，数据可能损坏")
    for barcode in barcodes:
        try:
            base64.b64decode(barcode.data)
            sys.stdout.write(barcode.data.decode())
            sys.stdout.flush()
        except:
            continue
