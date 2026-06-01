#include "crypto_utils.h"
#include "logger.h"

#include <windows.h>
#include <dpapi.h>

QString CryptoUtils::protect(const QString& plaintext)
{
    if (plaintext.isEmpty())
        return {};

    QByteArray utf8Data = plaintext.toUtf8();

    DATA_BLOB dataIn;
    dataIn.pbData = reinterpret_cast<BYTE*>(utf8Data.data());
    dataIn.cbData = static_cast<DWORD>(utf8Data.size());

    DATA_BLOB dataOut;
    ZeroMemory(&dataOut, sizeof(dataOut));

    if (!CryptProtectData(
            &dataIn,
            L"CampusAutoLogin",        // description — embedded in ciphertext
            nullptr,                    // no additional entropy
            nullptr, nullptr,           // reserved / no prompt
            CRYPTPROTECT_UI_FORBIDDEN,  // don't pop up UI
            &dataOut)) {
        DWORD err = GetLastError();
        Logger::instance().error(
            QString("DPAPI CryptProtectData failed (0x%1)").arg(err, 8, 16, QChar('0')));
        return {};
    }

    QByteArray encrypted(
        reinterpret_cast<const char*>(dataOut.pbData),
        static_cast<int>(dataOut.cbData));
    LocalFree(dataOut.pbData);

    return QString::fromLatin1(encrypted.toBase64());
}

QString CryptoUtils::unprotect(const QString& base64Ciphertext)
{
    if (base64Ciphertext.isEmpty())
        return {};

    QByteArray encrypted = QByteArray::fromBase64(base64Ciphertext.toLatin1());
    if (encrypted.isEmpty())
        return {};

    DATA_BLOB dataIn;
    dataIn.pbData = reinterpret_cast<BYTE*>(encrypted.data());
    dataIn.cbData = static_cast<DWORD>(encrypted.size());

    DATA_BLOB dataOut;
    ZeroMemory(&dataOut, sizeof(dataOut));

    if (!CryptUnprotectData(
            &dataIn,
            nullptr,                    // optional description out — not needed
            nullptr,                    // no additional entropy
            nullptr, nullptr,           // reserved / no prompt
            CRYPTPROTECT_UI_FORBIDDEN,
            &dataOut)) {
        DWORD err = GetLastError();
        Logger::instance().warning(
            QString("DPAPI CryptUnprotectData failed (0x%1) — data may be old-format or "
                    "from a different user account")
                .arg(err, 8, 16, QChar('0')));
        return {};
    }

    QByteArray decrypted(
        reinterpret_cast<const char*>(dataOut.pbData),
        static_cast<int>(dataOut.cbData));
    LocalFree(dataOut.pbData);

    return QString::fromUtf8(decrypted);
}
