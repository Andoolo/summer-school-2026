package com.volna.app.profile.data

import com.volna.app.auth.RequestCodeResult
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.ClientId
import com.volna.app.domain.model.Phone

fun ProfileClientDto.toDomain(): Client = Client(
    id = ClientId(id),
    name = name,
    phone = Phone(phone),
    createdAt = createdAt,
)

fun ProfileRequestCodeResponseDto.toDomain(): RequestCodeResult = RequestCodeResult(
    ttlSeconds = ttlSeconds,
    resendAfterSeconds = resendAfterSeconds,
)
