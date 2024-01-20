package com.httpsms

import timber.log.Timber
import java.security.MessageDigest
import java.util.Base64
import java.util.Random
import javax.crypto.Cipher
import javax.crypto.spec.IvParameterSpec
import javax.crypto.spec.SecretKeySpec

object Encrypter {
    private const val ALGORITHM = "AES/CFB/NoPadding"
    private const val IV_SIZE = 16

    fun decrypt(key: String, cipherText: String): String {
        val cipher = Cipher.getInstance(ALGORITHM)
        val cipherBytes = Base64.getDecoder().decode(cipherText)
        Timber.d("iv = ${Base64.getEncoder().encodeToString(cipherBytes.take(IV_SIZE).toByteArray())}")
        Timber.d("cipher = ${Base64.getEncoder().encodeToString(cipherBytes.drop(IV_SIZE).toByteArray())}")
        cipher.init(Cipher.DECRYPT_MODE, SecretKeySpec(hash(key), "AES"), IvParameterSpec(cipherBytes.take(IV_SIZE).toByteArray()))
        val plainText = cipher.doFinal(cipherBytes.drop(IV_SIZE).toByteArray())
        return String(plainText)
    }

    fun encrypt(key: String, inputText: String): String {
        val cipher = Cipher.getInstance(ALGORITHM)
        val iv = generateIv()
        cipher.init(Cipher.ENCRYPT_MODE, SecretKeySpec(hash(key),"AES"), IvParameterSpec(iv))
        val cipherBytes = iv + cipher.doFinal(inputText.toByteArray())
        return Base64.getEncoder().encodeToString(cipherBytes)
    }

    private fun generateIv(): ByteArray {
        val b = ByteArray(IV_SIZE)
        Random().nextBytes(b)
        return b
    }

    private fun hash(key: String): ByteArray {
        val bytes = key.toByteArray()
        val md = MessageDigest.getInstance("SHA-256")
        return md.digest(bytes)
    }
}
