package com.volna.app.profile

import com.volna.app.auth.RequestCodeResult
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.Phone

interface ProfileRepository {
    suspend fun getProfile(): Result<Client>
    suspend fun updateName(name: String): Result<Client>
    suspend fun deleteAccount(): Result<Unit>
    suspend fun requestPhoneChangeCode(newPhone: Phone): Result<RequestCodeResult>
    suspend fun confirmPhoneChange(newPhone: Phone, code: String): Result<Client>
}
