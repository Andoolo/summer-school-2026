package com.volna.app.core.phone

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class PhoneInputCoreTest {
    @Test
    fun `sanitize keeps ten local digits from formatted phone`() {
        assertEquals("9991234567", sanitizePhoneInput("+7 (999) 123-45-67"))
    }

    @Test
    fun `sanitize limits user input to ten digits`() {
        assertEquals("9991234567", sanitizePhoneInput("999123456789"))
    }

    @Test
    fun `normalize converts local digits to e164`() {
        assertEquals("+79991234567", normalizePhoneE164("9991234567"))
    }

    @Test
    fun `format displays local and e164 values with same mask`() {
        assertEquals("+7 (999) 123-45-67", formatPhoneNumber("9991234567"))
        assertEquals("+7 (999) 123-45-67", formatPhoneNumber("+79991234567"))
    }

    @Test
    fun `complete russian phone has ten local digits`() {
        assertTrue("9991234567".isRussianPhoneInputComplete())
    }
}
