package com.volna.app.auth.data

import com.volna.app.auth.RequestCodeResult
import com.volna.app.auth.VerifyCodeResult
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.ClientId
import com.volna.app.domain.model.Phone

fun RequestCodeResponseDto.toDomain(): RequestCodeResult = RequestCodeResult(
    ttlSeconds = ttlSeconds,
    resendAfterSeconds = resendAfterSeconds,
)

fun VerifyCodeResponseDto.toDomain(): VerifyCodeResult = VerifyCodeResult(
    token = token,
    client = client.toDomain(),
    isNew = isNew,
)

fun ClientDto.toDomain(): Client = Client(
    id = ClientId(id),
    name = name,
    phone = Phone(phone),
    createdAt = createdAt,
)
