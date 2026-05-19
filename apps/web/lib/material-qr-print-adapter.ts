export type MaterialQRCodePrintPayload = {
  materialName: string;
  materialSpec: string;
  materialLocation: string;
  qrCode: string;
};

export type MaterialQRCodePrintAdapter = {
  id: string;
  label: string;
  print: (payload: MaterialQRCodePrintPayload) => void;
};

export const browserMaterialQRCodePrintAdapter: MaterialQRCodePrintAdapter = {
  id: "browser",
  label: "浏览器打印",
  print: () => {
    window.print();
  },
};

export const materialQRCodePrintAdapters: MaterialQRCodePrintAdapter[] = [
  browserMaterialQRCodePrintAdapter,
];
