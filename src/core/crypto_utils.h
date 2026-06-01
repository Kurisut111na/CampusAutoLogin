#pragma once

#include <QString>

/// Thin wrapper around Windows DPAPI (CryptProtectData / CryptUnprotectData).
///
/// Encryption is bound to the current Windows user account — no key file needed,
/// no password to manage.  The ciphertext is base64-encoded for storage in JSON.
///
/// On a different machine or user account the ciphertext cannot be decrypted;
/// that's acceptable for this application (campus-network login tool).
class CryptoUtils
{
public:
    /// Encrypt @p plaintext with DPAPI, return base64 ciphertext.
    /// Returns an empty string on failure.
    static QString protect(const QString& plaintext);

    /// Decrypt @p base64Ciphertext with DPAPI, return plaintext.
    /// Returns an empty string on failure (wrong user, old-format data, etc.).
    static QString unprotect(const QString& base64Ciphertext);
};
