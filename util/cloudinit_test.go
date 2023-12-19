package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MergeCloudInitStrings", func() {
	var cloudinit1 string
	var cloudinit2 string
	var cloudinit3 string

	BeforeEach(func() {
		cloudinit1 = `#cloud-config
package_update: true
packages:
  - nginx
runcmd:
  - echo "hello world"
`
		cloudinit2 = `ssh_authorized_keys:
  - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAACAA... user@host`

		cloudinit3 = `#cloud-config
package_update: false
packages:
  - curl
runcmd:
  - echo "hello world 3"
`

	})
	It("Should show the right resulting cloud-init", func() {
		mergedCloudInit, err := MergeCloudInitData(cloudinit1, cloudinit2, cloudinit3)
		Expect(err).ToNot(HaveOccurred())
		mergedCloudInitString := string(mergedCloudInit)
		_, err = GinkgoWriter.Write(mergedCloudInit)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergedCloudInitString).To(Equal(`#cloud-config
package_update: false
packages:
- nginx
- curl
runcmd:
- echo "hello world"
- echo "hello world 3"
ssh_authorized_keys:
- ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAACAA... user@host
`))
	})
})

// TODO: Add more complex tests for MergeCloudInitData
var _ = Describe("MergeCloudInitDefaultHarvesterInit", func() {
	var cloudinit1 string
	var cloudinit2 string
	var cloudinit3 string

	BeforeEach(func() {
		cloudinit1 = `package_update: true
packages:
  - qemu-guest-agent
runcmd:
  - - systemctl
    - enable
    - --now
    - qemu-guest-agent.service
`
		cloudinit2 = `ssh_authorized_keys:
  - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAACAA... user@host`

		cloudinit3 = `## template: jinja
#cloud-config

write_files:
-   path: /etc/kubernetes/pki/ca.crt
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN CERTIFICATE-----
      MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
      cm5ldGVzMB4XDTIzMTIxMjEwNDkwOFoXDTMzMTIwOTEwNTQwOFowFTETMBEGA1UE
      AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANbP
      HiV15T+GX3arWSMRtDy31zyJ5jd5dCmudRPTtRJARVBCdy++pAGfDju4vBtZB9rN
      iNIwC5V5i6kc0c8i2pPINC7iM54NJydOIJPx6IDLwoC2p/1oVAD3IS6VquCe0ZxD
      srndmF7U58EO30YprDp44Jo9vt+i467l2A4jYDP6Hp+Sg695HXG14HKeK21l2a0B
      micqLx8UZMiMfiesIoZDOZHZMGyiQLf7JFONRrOwxztru2O7AtxWzzzNgjXGSaVn
      sMymcgl37BK30BYK8ziLF99vd8moOrSCOky+LiR3O877Z8vKZLaFHKCKVJYWcFOz
      0rphjrW0A6MppejxEHUCAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
      /wQIMAYBAf8CAQAwHQYDVR0OBBYEFBQqvC4QiQBKG9l592mci/aUvWdPMA0GCSqG
      SIb3DQEBCwUAA4IBAQDGowJ3xceYT1AV+5XrGABvcar6TMxxW6bgowrNbHhCDaqr
      twBAzhxuUd5CNx1LXvcstDq6mORMNd2LwLgPFsvIYrre+P8rTk53AEraxhwiG9Ep
      bP9C33SQNuZT2ALcoONmTQYeSV/7KbmV9O6HCqiF6v1bSqp7qDLLJE/00X8MDfBG
      q0CrdYoDS//CvK0FAjgHGoTneFjDVH093PL8/F9ydm0aHiHhLS2perLvnnqAOnK9
      NcuH+taB2KAFSaPB/OPYyWqZrKPj5X1GSyGcRkJXSVEHVmtQbIgt5qmC9uUSZ4zv
      0o1Ql/nuqQs7mz3EFthGi5CVxMX0j3N7Rjwk428t
      -----END CERTIFICATE-----
      
-   path: /etc/kubernetes/pki/ca.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEowIBAAKCAQEA1s8eJXXlP4ZfdqtZIxG0PLfXPInmN3l0Ka51E9O1EkBFUEJ3
      L76kAZ8OO7i8G1kH2s2I0jALlXmLqRzRzyLak8g0LuIzng0nJ04gk/HogMvCgLan
      /WhUAPchLpWq4J7RnEOyud2YXtTnwQ7fRimsOnjgmj2+36LjruXYDiNgM/oen5KD
      r3kdcbXgcp4rbWXZrQGaJyovHxRkyIx+J6wihkM5kdkwbKJAt/skU41Gs7DHO2u7
      Y7sC3FbPPM2CNcZJpWewzKZyCXfsErfQFgrzOIsX3293yag6tII6TL4uJHc7zvtn
      y8pktoUcoIpUlhZwU7PSumGOtbQDoyml6PEQdQIDAQABAoIBABEW/VEBpjF9oU6x
      py/REsPZ5HfeiMBVG1bNmGbxavB+yITwJMdZpXazjtBVjDGozaUswPvn8qP7vY7A
      yjhuj3E+dlhcirrCVSEdaB4dGuBUVa8j2Q2iJTzGbI9mPOgN+qMyB6Ad7ydsTNvh
      MQZF/nvQbh4XV343WWHqy1ukmNzJnhsyppbk/nDC1gpAbqumPsCBi/7Pp096OyZz
      QkEw/COqGGuOpaV0Ly/OgNapN5CUFLblPG0NBtOB3wQ+pW+LwGRytCRR8qfhEge/
      Z4YdnhNODjcgiin5fc2G0zR6x3U3O5WXAZBOYM783u0Axr/iYQ7H8d1F2PiYaGj3
      9pcza0ECgYEA4fBudefsgNKI6M9/FThnsCATBPwhfDd6L7AwnkYn4iX6uYx4lj/I
      iannCBR0F3KkzFHoXNeF2JHXRARyO7S2RrDln9K+f1xjEnJQi4ZQZhT5mJdlVlu6
      +YNR8zznIUp81o9JpvhH/Esz6f4sy1BCIxsIpcRBRVIBg9q6iJaDjwkCgYEA82OX
      DZr8+2ThzqK0mIK4Uw0QiMcWJVjYoUZkrXvSWeAf16FDFYGBNT0hVdS/hP2BKf+1
      SWvUWU0dhHBYcYFMgQfMkamTjHYavxK7jocIBP4M9F9fmM/YpOkUvxfLd0wlwmXE
      DzinxCqe+WQ4oFERChiCSz6cGfA/df5slUMCpQ0CgYBui1VwSL4VNW0ZA1S5TDSn
      HrpPiRDVFsuog3r2JXskEdL/b7QcRy7V9BP+hwtZ4ZSyBy06J5TsJkb9l3NQtRUt
      tyVSMilUZR5wCxBPg7LYj1CjkQda3ly38cFp0hV/21MDI240zGtkDGNlDCBchXMm
      e/aaLFCHGx10ptL3OzU5CQKBgQDagb16HHwk4kQLdG14QltjTGZctYe/Tc1mtMDs
      My79O0a7Gu8UHqk2d8Q2v4KVzdWpNAW4fdMtvRrT7NyqQm/Bo5PX7gsmXl3SzumN
      otLjUIWm2v0DPw57tznF+YHUf4uixCRJmg6cAbupoH1qCH2ot6o6DWKtss/2ic1I
      D9oO/QKBgGhwTicuIO8k0aopuaZumFtp0YwfTcS0TqAMseH9kBt2iGw48IqULKx8
      Ff4DcjYizdhi4vZSBjk/Hujq+Y5XdWnZBOw6W/tvi0T60AdVuX7go47yG5QXNYqT
      StmwOUN9iZbRvI6LevGvLwhRhi6BnGI5naVJOAqnz21umUWTw0ij
      -----END RSA PRIVATE KEY-----
      
-   path: /etc/kubernetes/pki/etcd/ca.crt
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN CERTIFICATE-----
      MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
      cm5ldGVzMB4XDTIzMTIxMjEwNDkwOFoXDTMzMTIwOTEwNTQwOFowFTETMBEGA1UE
      AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANhu
      XKmpQwkjcxt+QP2SmDg7DVOPHAuuotesRA9nFHQy9zsD/ZNT2IkP8iGCM5G2ZbHN
      v/2y7Wai57opP04BlYFuHg5shV+8N/sgZo2FbGjNH17pUSnlR/Sw2ocUxZ9qIZU0
      OJMF4fsRGxSk8nB9VwAmsUvGhv9pbhDmWZNDwkF6xNVXg0nk89uQyUDlatwFklIH
      EcRDBb6p2uGIR0J5A14/30kfUV5l7fLY/Z62x2W4TtIYbPePwhLzo9fGMBsrWGl4
      EX9NL9UQ4jbmrshV6m9TR0XSuCbwK+vFc9Bnn0mcGbOJmi8j/Ow603/ANM+HyUWe
      ueR0+7wEz/3WysoPm9UCAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
      /wQIMAYBAf8CAQAwHQYDVR0OBBYEFLW/QrKcxaj7Hb+vANsMh1L5xcbXMA0GCSqG
      SIb3DQEBCwUAA4IBAQAQWipunK8p7yQt2s/8mXS5G+nlLIrtkIMEWM+51k/ke3MA
      3GcIzevObJ0TFlNFL9aRihgZ05ei+uWuIbh5TlrEvM5l43kjR1Y+/4Ubr1z0bcm5
      Z5VZIQinhoBtG9nx6Sa0JsHIejs5YIn8gBBXeI7QlDT9yOZtmD99y8UwKb3oVUYc
      uBc/fcVsnwhb12xPNi1FBNyuQbVZBy+hyP8MPoG48e6C6JhQZkoelBYVIbG5rlIh
      xqm7C4M9Zrmu1DYuL/wZRy53Xqvznc8O+dWp1QbCx1okxpAIGvcTcxISXlG34zCj
      UIKfjg2wXGaSiRJ/FJIdDXrO3ekGZFTvfbdBrceZ
      -----END CERTIFICATE-----
      
-   path: /etc/kubernetes/pki/etcd/ca.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA2G5cqalDCSNzG35A/ZKYODsNU48cC66i16xED2cUdDL3OwP9
      k1PYiQ/yIYIzkbZlsc2//bLtZqLnuik/TgGVgW4eDmyFX7w3+yBmjYVsaM0fXulR
      KeVH9LDahxTFn2ohlTQ4kwXh+xEbFKTycH1XACaxS8aG/2luEOZZk0PCQXrE1VeD
      SeTz25DJQOVq3AWSUgcRxEMFvqna4YhHQnkDXj/fSR9RXmXt8tj9nrbHZbhO0hhs
      94/CEvOj18YwGytYaXgRf00v1RDiNuauyFXqb1NHRdK4JvAr68Vz0GefSZwZs4ma
      LyP87DrTf8A0z4fJRZ655HT7vATP/dbKyg+b1QIDAQABAoIBACOi8GEDPMV5b8+c
      F0lpZOUFXClhDAYkaC3I8J/0ohqL9cdi3dLvYF0ZIg5AaQtaFB6VuUIlvw9CTZOK
      jSDkA+D+57YKSl+8Fx+jcx9kU7hh5gNzuWiDlziEEkdhtTSNfiAaLCKROmdjpqjc
      jArXqIae2FyYwMu3aWcg9qjX5FlxdwgUQZzLUs/9wYHe/qdHyVPE30zYdsE7l7nz
      nkSPdKc6HkvIJY/R7U3WhNSEHSjCGmrydE2m+dBYp/Jk5Mx5Uj0ZeOXZjAdFy4J7
      u1XC4pPvF7XAedhyyEIHrBAvzDq6jHiqsO2xQNGkI/xgOA7ZiQkBgw5LxyCrTY78
      x27x5eECgYEA7iJpwq7Qmsyc1AmfYVPW89MfPdX3uy6m2f1zT68SAynJ2xRlK9u+
      kI06mSvJ5ddO3MCyXEFs48HEdj49WgCRCJ7j72fbNW3OPjsDGsj3QophNT8el98i
      B4jIDfBYdwdBQzsQ7I+8bMsVtrUQjDMheWYh1Ra2iULBl+KtraAbDzkCgYEA6Ksd
      6xfj4+F2KgiAsWSJ/M6cmV1BH5Uy7QWABqLOMV4T0ARF9usSqcS0493rZOqMMkAp
      IzCE2+nVbN56jNxfjjKi++/E3u+UX4Lq0W38sY3XP2OfwdbRtSbaPNAxBkZSaUQu
      VqSwyQVXjrMZpRMS+WNaimv7wyWIP8MLKjfllX0CgYBd9CPoFNLnEG2b1wQUAWEg
      qB5+ZiospvZbsXzKZpdjuhwTHNPh3vwryhzhi/5HeZB61mhIr+OHZM7fnCTWmrye
      OxpRPZemV+F0ehH6gmnTzgcWXAX1A6tIb7YGkdpFdA5SuT4vJ3K/Nc0mXf/eYNoH
      LL2SdjikpTr+cwf1JeMnOQKBgQC1Z10fQ/QpY0s3AIQeSw4O7qRIKt4wmqonBMfJ
      5Luw3/HAmORX3PYjKTwEAa2bdAe00jOAvT6JG6qMhHW2R8e03aQXm9y6GL9tLGya
      tw9y++0b/je78Rp2DAHRslzW0JNGgaNDaIpxYNngZ6GSA+oiSSV5kTGs+CFf3Vli
      JEy7HQKBgQCW7pS9saWTl2J8auFpHCu5ZtTz1AmPIawlbRJiwnGrOTUJ6AOG30m0
      XDYiIt4GEzoeSnxEmVKILQF+J4hnc1PzKgugfSYUEcGX260EYJ5XS/fYKznSWV2W
      x4Qr5XLezfjiSF5gUruyzTADYZUYD22RziiDqMWxuQ5YGlgdsj0D8g==
      -----END RSA PRIVATE KEY-----
      
-   path: /etc/kubernetes/pki/front-proxy-ca.crt
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN CERTIFICATE-----
      MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
      cm5ldGVzMB4XDTIzMTIxMjEwNDkwOFoXDTMzMTIwOTEwNTQwOFowFTETMBEGA1UE
      AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMqy
      XOqb+o4L94Ci6tyqoYTqDQrT6xsPs5p/383yAcsfVeBfrkaGOor/jT+DSASCddD3
      d3a1vj7gv5hFWjkfcq9e4S2Hs8dMcAGGoH6/+GWVOs3vXTy04KLEV71r1BYStUVx
      1pmMz9cyILcbF+b9N4VxR6L7PpM94utsUPpoOxBFe3WghWTtktzkGLJSAIND+Oz9
      KSplahwo7T8Gh/YzH8GQ8jluI92VxOGNEXwsVPrkkD3h7IaCFf8C3UAbIMP7IDSG
      N6xRoi+j5hYjcPlfwpJZmJtk0qgzLJqgLaXGQs/wsV1KktHxcGbTqfwY+4Dn+xY7
      3w1LkeCYTAkNIGqFEO8CAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
      /wQIMAYBAf8CAQAwHQYDVR0OBBYEFLhHaD7IpU8XjVM/JaumVJWoaTh4MA0GCSqG
      SIb3DQEBCwUAA4IBAQB29LsJ5x7bLoyuhBtRshn/qUCkfkAYp0kRISnxeUuEd5yi
      6OMpT+45XEqqFhJipmq9Lwfc+LBWh7fbq7UV78u5bbokOcGOZuaZy6OtU4gKG+PP
      MgBVAfE+e2ILL51sdq1LF+6XwmwPO0h5SaFFpTqMagjAOxzHXLNURAM2etqll2OA
      dnHl19jwBzCPZo+vAb5BehrtekZm7Bc9E4apeimhrNht8wplBZop1exPYpA68LKi
      3WBF4hdRvD4M1oESG3DaZXhUSxy1+eVslN2WIfb9+Ok3TmxYFgMZ9Jrqn82U5ccL
      SXGmBwdl1U3v86eLaO99rY2EW7dBGNZV7FNuq0ft
      -----END CERTIFICATE-----
      
-   path: /etc/kubernetes/pki/front-proxy-ca.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEAyrJc6pv6jgv3gKLq3KqhhOoNCtPrGw+zmn/fzfIByx9V4F+u
      RoY6iv+NP4NIBIJ10Pd3drW+PuC/mEVaOR9yr17hLYezx0xwAYagfr/4ZZU6ze9d
      PLTgosRXvWvUFhK1RXHWmYzP1zIgtxsX5v03hXFHovs+kz3i62xQ+mg7EEV7daCF
      ZO2S3OQYslIAg0P47P0pKmVqHCjtPwaH9jMfwZDyOW4j3ZXE4Y0RfCxU+uSQPeHs
      hoIV/wLdQBsgw/sgNIY3rFGiL6PmFiNw+V/CklmYm2TSqDMsmqAtpcZCz/CxXUqS
      0fFwZtOp/Bj7gOf7FjvfDUuR4JhMCQ0gaoUQ7wIDAQABAoIBAQCLZF+Lo5qJxub9
      GoyjFeCftAkmEhhTctfTfu7dBPmAw1reQ05pB3QJFLcBH3oOR91XyGbqRw++0/ZO
      dBsYv2yx93CpS/IxM3qvQfLrV38t9JMM/fhDgCwfIyEnjZi7WUA5spCe5fwkhD+F
      TGeCnU5qQT2/ckJVJbEAr2t82OMNS1Efubl7EFMOABuMsxavorIKdKkI+ZvGzyS3
      Gkk2yEr55M+bpK6dd56Hfu12CSCKwVYLaKAmOKO+8rqZVKAi09Ig8ki2TgGopFxq
      ZR9WYguQZg20BEXe+EPgMmSqPDA9y8w7bidteYHNiq25C8tu90oQHI45c89ddFRz
      ttR/agHhAoGBANVDzTcxlq0t+8UxabtYdaySLgAsOi7n/XvjqFrlkrYW/6IeWwXN
      xgJPst1FvXbJ+Luar7BjK3nf75P7hgQ5l6EpslLPwvTy/AsFoRh0xbtA/rh5QpC6
      7W16sGkr/OQd8LIQlEKKug/YiKUi+pFNt6TQQWXkrLZY2Q0wIT+1iYFRAoGBAPNQ
      blss3u5fP225bs2UClV7XqT0Fq0VVRwP2wZKqe8h5HH2dWZtKIYzPCohab8vBOAr
      cWDYH2Gz/WoWb3deqD5kTiOZ+dJ3ggaAkI8hSYj6Cpu7cpqzdABL58PQMb8FFuTX
      D+WBGQPZbvNDi56G8dOTaawXwYH2tJWW0Fm0at4/AoGBALzuOwA5ix3SzeftFZkm
      DeGbAuueQtFJLnQxw/T6ypVMHJ23vLWQjWmAx5llbiqtVRCGQjzGLj7jFzCHNDvL
      9buN3++jJTixhn4RN50d3go80ywEKOdk4nAJr/0MPhatO43USDQHCDx/fNam/Un6
      isWUxUsKYcONRIR9bgctwSpxAoGAT7TQggO///ypzasKVkQh4oDor0baytaLLAcx
      q+z3oEPND1w6d1RZCyVrly2c86lWgo0Yti32kc4hvQgeec9DdDTtuBHv2feWW8Tw
      FkNEUKAAq6WLVIxm+tXi1a21LitfpZWiOn/BDxbCluRQr5zrSXEoE90wYf/MhpiC
      JnDI9YcCgYB1jGq6Xr5qJMaqjpOEPPzHy+5PYlBmHOEJA9k9ZeT07g3yc9yhOFpT
      dtP9haR01Yr1dHmIb3OmQOIV8OUnJTexMzm8AycXvtpc2mi6vnNiS7MZn2mHahs0
      LKijtTtPJ3to+Z3pPy6rGEXZ/RI4PdsWBuIaz8uqpl6WTfqtFbrvBw==
      -----END RSA PRIVATE KEY-----
      
-   path: /etc/kubernetes/pki/sa.pub
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtpb9MFW1QRfDlalmneSZ
      UD5rZ+ms7kFjVC7ixR+9Tf7ijdWAMGt8Sw5uF25xVvb9aPWmDqQQQVQyBt1xfWqn
      6LX7f0xB/GGRgZavlmt0HRGLAXhW9isqM4Um0LHnB4HPRok7OYcK4PUittr1wAKe
      rR+Ou/xlOWOCFt17sYepF0j2WQlEy9a1TErbEJFJp5TffHhiXd8IDM/rLEZ6uG33
      FyspsmjN6vVB+5dkZnJfA8UpZ1iAllvmBFI+v2IM7NRXiDm3AdaM8qabiSX2HAeV
      qOXLhff8F6UTNyF8uCX/OaBfAZdON5oD+Gn/RjEG8reTZl+TAAPmLQPKLfcnhq40
      uwIDAQAB
      -----END PUBLIC KEY-----
      
-   path: /etc/kubernetes/pki/sa.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEowIBAAKCAQEAtpb9MFW1QRfDlalmneSZUD5rZ+ms7kFjVC7ixR+9Tf7ijdWA
      MGt8Sw5uF25xVvb9aPWmDqQQQVQyBt1xfWqn6LX7f0xB/GGRgZavlmt0HRGLAXhW
      9isqM4Um0LHnB4HPRok7OYcK4PUittr1wAKerR+Ou/xlOWOCFt17sYepF0j2WQlE
      y9a1TErbEJFJp5TffHhiXd8IDM/rLEZ6uG33FyspsmjN6vVB+5dkZnJfA8UpZ1iA
      llvmBFI+v2IM7NRXiDm3AdaM8qabiSX2HAeVqOXLhff8F6UTNyF8uCX/OaBfAZdO
      N5oD+Gn/RjEG8reTZl+TAAPmLQPKLfcnhq40uwIDAQABAoIBACclmiUZyyGomatl
      xXWGxIQazeZaiFQQut4aq03+LxUg16v3IWPAN8bT0jC94hj2HYC6Yh7zd/S5u3wT
      UDjGfDd9hO1XCTK2LH8vMng6k4uD7lyjU2m1+XdQTfEio1jNsQX7eDIuTNvMUuQH
      b/b52NFfWbfeNkmmlwaV9+YpIsy13++KJzIn/NlCHkOiMvtHTvf3ZUsPr9Kr6STJ
      7tt0+V9qPcajLckMMKqUoKgNcYAXkWgtM2BQteewPDSNXtxcviBVxHmsfyjuK0bN
      4rXOLhDua4bkbb6FslJXYpyiz7/cLqJygmGRDW9mf+I0QSYFstwQRzxchcS57upJ
      5a18GLECgYEA6oimH0I2b3rghSpS84CuEJcTjyUXvf1D5HeI5vHEvo0l9K1/E+of
      SLz3w6qsUBOFdSF0Vm/k5S75Sska6EGhtIe+agbhCNdULRjBvz5LToIDqovBcGq3
      sBnJxW4LQ2PQM68OXN2Wl4UEWF9Umis5CZsl+i81GiAVnnNjgh8XYkkCgYEAx00+
      KorTQBe5c9/uYPvYEHG7FsVSprMWsm3GWOJNhqgW27EkKqGRPLu2UylOeYe0eyj3
      UkKFZHxl6Kt+xxAH50lcZ3aT6wUf9cXWVLcOy7IR+5FAkqeGvgKexbNZkLKiArfe
      SQo5FF7GIu0gP8W/fgv0r5itrAQwxOBnTQQBnuMCgYEAjHRpiC7PCtQ7wYQnSUy2
      8ZiITiGYpl8WWax8gFIp0TQWlwGQKQz8z0Lb3oJHz2zhb9QpJ9q66cXH5dGqG42y
      mbrxfe3Attq9voQlA7L6xnl2WJx5rCk8+Gl5PJM6i5ErDsi3gUXy+arff00YDXv1
      HJudksbStmKgj9Pqs/KKvoECgYAWmePa3zNlqUsWoOZfiS/PbZZR1r6wuM5yHZDI
      s6EnDBjLgSMg0oGt6XuboquLjKAi91pUscZ+xrynzgrqeB7tU5xu/zt3A3XEYVMU
      +E1tPBxd8vLnrqfRFGr88IHPrvJAbKmAjvA6JyVBALMPiFVW7fQplZ7cSv1c1jXg
      vfuREQKBgE77qPlhDBwRqMB8H4A7Ai8lj1OrYbnqau4hGto3CUa5c8HKTx8LBvod
      jb3PUVkXrXmITZi77LH/wpV+ktuxmox+XRbfNZRZjACz2sts+She3A+nVa+jTPAa
      BMgC7znkT50NXccDayxmS62oqLbnOaGxKpk4h8UfKv5qh+NbEfiM
      -----END RSA PRIVATE KEY-----
      
-   path: /run/kubeadm/kubeadm.yaml
    owner: root:root
    permissions: '0640'
    content: |
      ---
      apiServer: {}
      apiVersion: kubeadm.k8s.io/v1beta3
      clusterName: test
      controlPlaneEndpoint: 127.0.0.1:6443
      controllerManager: {}
      dns: {}
      etcd: {}
      kind: ClusterConfiguration
      kubernetesVersion: v1.26.6
      networking:
        dnsDomain: cluster.local
        podSubnet: 192.168.0.0/16
      scheduler: {}
      
      ---
      apiVersion: kubeadm.k8s.io/v1beta3
      kind: InitConfiguration
      localAPIEndpoint: {}
      nodeRegistration:
        imagePullPolicy: IfNotPresent
        taints: null
      
-   path: /run/cluster-api/placeholder
    owner: root:root
    permissions: '0640'
    content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
runcmd:
  - 'kubeadm init --config /run/kubeadm/kubeadm.yaml  && echo success > /run/cluster-api/bootstrap-success.complete'`

	})
	It("Should show the right resulting cloud-init with a basic init for Harvester", func() {
		mergedCloudInit, err := MergeCloudInitData(cloudinit1, cloudinit2, cloudinit3)
		Expect(err).ToNot(HaveOccurred())
		mergedCloudInitString := string(mergedCloudInit)
		_, err = GinkgoWriter.Write(mergedCloudInit)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergedCloudInitString).To(Equal(`#cloud-config
package_update: true
packages:
- qemu-guest-agent
runcmd:
- - systemctl
  - enable
  - --now
  - qemu-guest-agent.service
- 'kubeadm init --config /run/kubeadm/kubeadm.yaml  && echo success > /run/cluster-api/bootstrap-success.complete'
ssh_authorized_keys:
- ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAACAA... user@host
write_files:
-   path: /etc/kubernetes/pki/ca.crt
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN CERTIFICATE-----
      MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
      cm5ldGVzMB4XDTIzMTIxMjEwNDkwOFoXDTMzMTIwOTEwNTQwOFowFTETMBEGA1UE
      AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANbP
      HiV15T+GX3arWSMRtDy31zyJ5jd5dCmudRPTtRJARVBCdy++pAGfDju4vBtZB9rN
      iNIwC5V5i6kc0c8i2pPINC7iM54NJydOIJPx6IDLwoC2p/1oVAD3IS6VquCe0ZxD
      srndmF7U58EO30YprDp44Jo9vt+i467l2A4jYDP6Hp+Sg695HXG14HKeK21l2a0B
      micqLx8UZMiMfiesIoZDOZHZMGyiQLf7JFONRrOwxztru2O7AtxWzzzNgjXGSaVn
      sMymcgl37BK30BYK8ziLF99vd8moOrSCOky+LiR3O877Z8vKZLaFHKCKVJYWcFOz
      0rphjrW0A6MppejxEHUCAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
      /wQIMAYBAf8CAQAwHQYDVR0OBBYEFBQqvC4QiQBKG9l592mci/aUvWdPMA0GCSqG
      SIb3DQEBCwUAA4IBAQDGowJ3xceYT1AV+5XrGABvcar6TMxxW6bgowrNbHhCDaqr
      twBAzhxuUd5CNx1LXvcstDq6mORMNd2LwLgPFsvIYrre+P8rTk53AEraxhwiG9Ep
      bP9C33SQNuZT2ALcoONmTQYeSV/7KbmV9O6HCqiF6v1bSqp7qDLLJE/00X8MDfBG
      q0CrdYoDS//CvK0FAjgHGoTneFjDVH093PL8/F9ydm0aHiHhLS2perLvnnqAOnK9
      NcuH+taB2KAFSaPB/OPYyWqZrKPj5X1GSyGcRkJXSVEHVmtQbIgt5qmC9uUSZ4zv
      0o1Ql/nuqQs7mz3EFthGi5CVxMX0j3N7Rjwk428t
      -----END CERTIFICATE-----
      
-   path: /etc/kubernetes/pki/ca.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEowIBAAKCAQEA1s8eJXXlP4ZfdqtZIxG0PLfXPInmN3l0Ka51E9O1EkBFUEJ3
      L76kAZ8OO7i8G1kH2s2I0jALlXmLqRzRzyLak8g0LuIzng0nJ04gk/HogMvCgLan
      /WhUAPchLpWq4J7RnEOyud2YXtTnwQ7fRimsOnjgmj2+36LjruXYDiNgM/oen5KD
      r3kdcbXgcp4rbWXZrQGaJyovHxRkyIx+J6wihkM5kdkwbKJAt/skU41Gs7DHO2u7
      Y7sC3FbPPM2CNcZJpWewzKZyCXfsErfQFgrzOIsX3293yag6tII6TL4uJHc7zvtn
      y8pktoUcoIpUlhZwU7PSumGOtbQDoyml6PEQdQIDAQABAoIBABEW/VEBpjF9oU6x
      py/REsPZ5HfeiMBVG1bNmGbxavB+yITwJMdZpXazjtBVjDGozaUswPvn8qP7vY7A
      yjhuj3E+dlhcirrCVSEdaB4dGuBUVa8j2Q2iJTzGbI9mPOgN+qMyB6Ad7ydsTNvh
      MQZF/nvQbh4XV343WWHqy1ukmNzJnhsyppbk/nDC1gpAbqumPsCBi/7Pp096OyZz
      QkEw/COqGGuOpaV0Ly/OgNapN5CUFLblPG0NBtOB3wQ+pW+LwGRytCRR8qfhEge/
      Z4YdnhNODjcgiin5fc2G0zR6x3U3O5WXAZBOYM783u0Axr/iYQ7H8d1F2PiYaGj3
      9pcza0ECgYEA4fBudefsgNKI6M9/FThnsCATBPwhfDd6L7AwnkYn4iX6uYx4lj/I
      iannCBR0F3KkzFHoXNeF2JHXRARyO7S2RrDln9K+f1xjEnJQi4ZQZhT5mJdlVlu6
      +YNR8zznIUp81o9JpvhH/Esz6f4sy1BCIxsIpcRBRVIBg9q6iJaDjwkCgYEA82OX
      DZr8+2ThzqK0mIK4Uw0QiMcWJVjYoUZkrXvSWeAf16FDFYGBNT0hVdS/hP2BKf+1
      SWvUWU0dhHBYcYFMgQfMkamTjHYavxK7jocIBP4M9F9fmM/YpOkUvxfLd0wlwmXE
      DzinxCqe+WQ4oFERChiCSz6cGfA/df5slUMCpQ0CgYBui1VwSL4VNW0ZA1S5TDSn
      HrpPiRDVFsuog3r2JXskEdL/b7QcRy7V9BP+hwtZ4ZSyBy06J5TsJkb9l3NQtRUt
      tyVSMilUZR5wCxBPg7LYj1CjkQda3ly38cFp0hV/21MDI240zGtkDGNlDCBchXMm
      e/aaLFCHGx10ptL3OzU5CQKBgQDagb16HHwk4kQLdG14QltjTGZctYe/Tc1mtMDs
      My79O0a7Gu8UHqk2d8Q2v4KVzdWpNAW4fdMtvRrT7NyqQm/Bo5PX7gsmXl3SzumN
      otLjUIWm2v0DPw57tznF+YHUf4uixCRJmg6cAbupoH1qCH2ot6o6DWKtss/2ic1I
      D9oO/QKBgGhwTicuIO8k0aopuaZumFtp0YwfTcS0TqAMseH9kBt2iGw48IqULKx8
      Ff4DcjYizdhi4vZSBjk/Hujq+Y5XdWnZBOw6W/tvi0T60AdVuX7go47yG5QXNYqT
      StmwOUN9iZbRvI6LevGvLwhRhi6BnGI5naVJOAqnz21umUWTw0ij
      -----END RSA PRIVATE KEY-----
      
-   path: /etc/kubernetes/pki/etcd/ca.crt
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN CERTIFICATE-----
      MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
      cm5ldGVzMB4XDTIzMTIxMjEwNDkwOFoXDTMzMTIwOTEwNTQwOFowFTETMBEGA1UE
      AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANhu
      XKmpQwkjcxt+QP2SmDg7DVOPHAuuotesRA9nFHQy9zsD/ZNT2IkP8iGCM5G2ZbHN
      v/2y7Wai57opP04BlYFuHg5shV+8N/sgZo2FbGjNH17pUSnlR/Sw2ocUxZ9qIZU0
      OJMF4fsRGxSk8nB9VwAmsUvGhv9pbhDmWZNDwkF6xNVXg0nk89uQyUDlatwFklIH
      EcRDBb6p2uGIR0J5A14/30kfUV5l7fLY/Z62x2W4TtIYbPePwhLzo9fGMBsrWGl4
      EX9NL9UQ4jbmrshV6m9TR0XSuCbwK+vFc9Bnn0mcGbOJmi8j/Ow603/ANM+HyUWe
      ueR0+7wEz/3WysoPm9UCAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
      /wQIMAYBAf8CAQAwHQYDVR0OBBYEFLW/QrKcxaj7Hb+vANsMh1L5xcbXMA0GCSqG
      SIb3DQEBCwUAA4IBAQAQWipunK8p7yQt2s/8mXS5G+nlLIrtkIMEWM+51k/ke3MA
      3GcIzevObJ0TFlNFL9aRihgZ05ei+uWuIbh5TlrEvM5l43kjR1Y+/4Ubr1z0bcm5
      Z5VZIQinhoBtG9nx6Sa0JsHIejs5YIn8gBBXeI7QlDT9yOZtmD99y8UwKb3oVUYc
      uBc/fcVsnwhb12xPNi1FBNyuQbVZBy+hyP8MPoG48e6C6JhQZkoelBYVIbG5rlIh
      xqm7C4M9Zrmu1DYuL/wZRy53Xqvznc8O+dWp1QbCx1okxpAIGvcTcxISXlG34zCj
      UIKfjg2wXGaSiRJ/FJIdDXrO3ekGZFTvfbdBrceZ
      -----END CERTIFICATE-----
      
-   path: /etc/kubernetes/pki/etcd/ca.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA2G5cqalDCSNzG35A/ZKYODsNU48cC66i16xED2cUdDL3OwP9
      k1PYiQ/yIYIzkbZlsc2//bLtZqLnuik/TgGVgW4eDmyFX7w3+yBmjYVsaM0fXulR
      KeVH9LDahxTFn2ohlTQ4kwXh+xEbFKTycH1XACaxS8aG/2luEOZZk0PCQXrE1VeD
      SeTz25DJQOVq3AWSUgcRxEMFvqna4YhHQnkDXj/fSR9RXmXt8tj9nrbHZbhO0hhs
      94/CEvOj18YwGytYaXgRf00v1RDiNuauyFXqb1NHRdK4JvAr68Vz0GefSZwZs4ma
      LyP87DrTf8A0z4fJRZ655HT7vATP/dbKyg+b1QIDAQABAoIBACOi8GEDPMV5b8+c
      F0lpZOUFXClhDAYkaC3I8J/0ohqL9cdi3dLvYF0ZIg5AaQtaFB6VuUIlvw9CTZOK
      jSDkA+D+57YKSl+8Fx+jcx9kU7hh5gNzuWiDlziEEkdhtTSNfiAaLCKROmdjpqjc
      jArXqIae2FyYwMu3aWcg9qjX5FlxdwgUQZzLUs/9wYHe/qdHyVPE30zYdsE7l7nz
      nkSPdKc6HkvIJY/R7U3WhNSEHSjCGmrydE2m+dBYp/Jk5Mx5Uj0ZeOXZjAdFy4J7
      u1XC4pPvF7XAedhyyEIHrBAvzDq6jHiqsO2xQNGkI/xgOA7ZiQkBgw5LxyCrTY78
      x27x5eECgYEA7iJpwq7Qmsyc1AmfYVPW89MfPdX3uy6m2f1zT68SAynJ2xRlK9u+
      kI06mSvJ5ddO3MCyXEFs48HEdj49WgCRCJ7j72fbNW3OPjsDGsj3QophNT8el98i
      B4jIDfBYdwdBQzsQ7I+8bMsVtrUQjDMheWYh1Ra2iULBl+KtraAbDzkCgYEA6Ksd
      6xfj4+F2KgiAsWSJ/M6cmV1BH5Uy7QWABqLOMV4T0ARF9usSqcS0493rZOqMMkAp
      IzCE2+nVbN56jNxfjjKi++/E3u+UX4Lq0W38sY3XP2OfwdbRtSbaPNAxBkZSaUQu
      VqSwyQVXjrMZpRMS+WNaimv7wyWIP8MLKjfllX0CgYBd9CPoFNLnEG2b1wQUAWEg
      qB5+ZiospvZbsXzKZpdjuhwTHNPh3vwryhzhi/5HeZB61mhIr+OHZM7fnCTWmrye
      OxpRPZemV+F0ehH6gmnTzgcWXAX1A6tIb7YGkdpFdA5SuT4vJ3K/Nc0mXf/eYNoH
      LL2SdjikpTr+cwf1JeMnOQKBgQC1Z10fQ/QpY0s3AIQeSw4O7qRIKt4wmqonBMfJ
      5Luw3/HAmORX3PYjKTwEAa2bdAe00jOAvT6JG6qMhHW2R8e03aQXm9y6GL9tLGya
      tw9y++0b/je78Rp2DAHRslzW0JNGgaNDaIpxYNngZ6GSA+oiSSV5kTGs+CFf3Vli
      JEy7HQKBgQCW7pS9saWTl2J8auFpHCu5ZtTz1AmPIawlbRJiwnGrOTUJ6AOG30m0
      XDYiIt4GEzoeSnxEmVKILQF+J4hnc1PzKgugfSYUEcGX260EYJ5XS/fYKznSWV2W
      x4Qr5XLezfjiSF5gUruyzTADYZUYD22RziiDqMWxuQ5YGlgdsj0D8g==
      -----END RSA PRIVATE KEY-----
      
-   path: /etc/kubernetes/pki/front-proxy-ca.crt
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN CERTIFICATE-----
      MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
      cm5ldGVzMB4XDTIzMTIxMjEwNDkwOFoXDTMzMTIwOTEwNTQwOFowFTETMBEGA1UE
      AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMqy
      XOqb+o4L94Ci6tyqoYTqDQrT6xsPs5p/383yAcsfVeBfrkaGOor/jT+DSASCddD3
      d3a1vj7gv5hFWjkfcq9e4S2Hs8dMcAGGoH6/+GWVOs3vXTy04KLEV71r1BYStUVx
      1pmMz9cyILcbF+b9N4VxR6L7PpM94utsUPpoOxBFe3WghWTtktzkGLJSAIND+Oz9
      KSplahwo7T8Gh/YzH8GQ8jluI92VxOGNEXwsVPrkkD3h7IaCFf8C3UAbIMP7IDSG
      N6xRoi+j5hYjcPlfwpJZmJtk0qgzLJqgLaXGQs/wsV1KktHxcGbTqfwY+4Dn+xY7
      3w1LkeCYTAkNIGqFEO8CAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
      /wQIMAYBAf8CAQAwHQYDVR0OBBYEFLhHaD7IpU8XjVM/JaumVJWoaTh4MA0GCSqG
      SIb3DQEBCwUAA4IBAQB29LsJ5x7bLoyuhBtRshn/qUCkfkAYp0kRISnxeUuEd5yi
      6OMpT+45XEqqFhJipmq9Lwfc+LBWh7fbq7UV78u5bbokOcGOZuaZy6OtU4gKG+PP
      MgBVAfE+e2ILL51sdq1LF+6XwmwPO0h5SaFFpTqMagjAOxzHXLNURAM2etqll2OA
      dnHl19jwBzCPZo+vAb5BehrtekZm7Bc9E4apeimhrNht8wplBZop1exPYpA68LKi
      3WBF4hdRvD4M1oESG3DaZXhUSxy1+eVslN2WIfb9+Ok3TmxYFgMZ9Jrqn82U5ccL
      SXGmBwdl1U3v86eLaO99rY2EW7dBGNZV7FNuq0ft
      -----END CERTIFICATE-----
      
-   path: /etc/kubernetes/pki/front-proxy-ca.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEAyrJc6pv6jgv3gKLq3KqhhOoNCtPrGw+zmn/fzfIByx9V4F+u
      RoY6iv+NP4NIBIJ10Pd3drW+PuC/mEVaOR9yr17hLYezx0xwAYagfr/4ZZU6ze9d
      PLTgosRXvWvUFhK1RXHWmYzP1zIgtxsX5v03hXFHovs+kz3i62xQ+mg7EEV7daCF
      ZO2S3OQYslIAg0P47P0pKmVqHCjtPwaH9jMfwZDyOW4j3ZXE4Y0RfCxU+uSQPeHs
      hoIV/wLdQBsgw/sgNIY3rFGiL6PmFiNw+V/CklmYm2TSqDMsmqAtpcZCz/CxXUqS
      0fFwZtOp/Bj7gOf7FjvfDUuR4JhMCQ0gaoUQ7wIDAQABAoIBAQCLZF+Lo5qJxub9
      GoyjFeCftAkmEhhTctfTfu7dBPmAw1reQ05pB3QJFLcBH3oOR91XyGbqRw++0/ZO
      dBsYv2yx93CpS/IxM3qvQfLrV38t9JMM/fhDgCwfIyEnjZi7WUA5spCe5fwkhD+F
      TGeCnU5qQT2/ckJVJbEAr2t82OMNS1Efubl7EFMOABuMsxavorIKdKkI+ZvGzyS3
      Gkk2yEr55M+bpK6dd56Hfu12CSCKwVYLaKAmOKO+8rqZVKAi09Ig8ki2TgGopFxq
      ZR9WYguQZg20BEXe+EPgMmSqPDA9y8w7bidteYHNiq25C8tu90oQHI45c89ddFRz
      ttR/agHhAoGBANVDzTcxlq0t+8UxabtYdaySLgAsOi7n/XvjqFrlkrYW/6IeWwXN
      xgJPst1FvXbJ+Luar7BjK3nf75P7hgQ5l6EpslLPwvTy/AsFoRh0xbtA/rh5QpC6
      7W16sGkr/OQd8LIQlEKKug/YiKUi+pFNt6TQQWXkrLZY2Q0wIT+1iYFRAoGBAPNQ
      blss3u5fP225bs2UClV7XqT0Fq0VVRwP2wZKqe8h5HH2dWZtKIYzPCohab8vBOAr
      cWDYH2Gz/WoWb3deqD5kTiOZ+dJ3ggaAkI8hSYj6Cpu7cpqzdABL58PQMb8FFuTX
      D+WBGQPZbvNDi56G8dOTaawXwYH2tJWW0Fm0at4/AoGBALzuOwA5ix3SzeftFZkm
      DeGbAuueQtFJLnQxw/T6ypVMHJ23vLWQjWmAx5llbiqtVRCGQjzGLj7jFzCHNDvL
      9buN3++jJTixhn4RN50d3go80ywEKOdk4nAJr/0MPhatO43USDQHCDx/fNam/Un6
      isWUxUsKYcONRIR9bgctwSpxAoGAT7TQggO///ypzasKVkQh4oDor0baytaLLAcx
      q+z3oEPND1w6d1RZCyVrly2c86lWgo0Yti32kc4hvQgeec9DdDTtuBHv2feWW8Tw
      FkNEUKAAq6WLVIxm+tXi1a21LitfpZWiOn/BDxbCluRQr5zrSXEoE90wYf/MhpiC
      JnDI9YcCgYB1jGq6Xr5qJMaqjpOEPPzHy+5PYlBmHOEJA9k9ZeT07g3yc9yhOFpT
      dtP9haR01Yr1dHmIb3OmQOIV8OUnJTexMzm8AycXvtpc2mi6vnNiS7MZn2mHahs0
      LKijtTtPJ3to+Z3pPy6rGEXZ/RI4PdsWBuIaz8uqpl6WTfqtFbrvBw==
      -----END RSA PRIVATE KEY-----
      
-   path: /etc/kubernetes/pki/sa.pub
    owner: root:root
    permissions: '0640'
    content: |
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtpb9MFW1QRfDlalmneSZ
      UD5rZ+ms7kFjVC7ixR+9Tf7ijdWAMGt8Sw5uF25xVvb9aPWmDqQQQVQyBt1xfWqn
      6LX7f0xB/GGRgZavlmt0HRGLAXhW9isqM4Um0LHnB4HPRok7OYcK4PUittr1wAKe
      rR+Ou/xlOWOCFt17sYepF0j2WQlEy9a1TErbEJFJp5TffHhiXd8IDM/rLEZ6uG33
      FyspsmjN6vVB+5dkZnJfA8UpZ1iAllvmBFI+v2IM7NRXiDm3AdaM8qabiSX2HAeV
      qOXLhff8F6UTNyF8uCX/OaBfAZdON5oD+Gn/RjEG8reTZl+TAAPmLQPKLfcnhq40
      uwIDAQAB
      -----END PUBLIC KEY-----
      
-   path: /etc/kubernetes/pki/sa.key
    owner: root:root
    permissions: '0600'
    content: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEowIBAAKCAQEAtpb9MFW1QRfDlalmneSZUD5rZ+ms7kFjVC7ixR+9Tf7ijdWA
      MGt8Sw5uF25xVvb9aPWmDqQQQVQyBt1xfWqn6LX7f0xB/GGRgZavlmt0HRGLAXhW
      9isqM4Um0LHnB4HPRok7OYcK4PUittr1wAKerR+Ou/xlOWOCFt17sYepF0j2WQlE
      y9a1TErbEJFJp5TffHhiXd8IDM/rLEZ6uG33FyspsmjN6vVB+5dkZnJfA8UpZ1iA
      llvmBFI+v2IM7NRXiDm3AdaM8qabiSX2HAeVqOXLhff8F6UTNyF8uCX/OaBfAZdO
      N5oD+Gn/RjEG8reTZl+TAAPmLQPKLfcnhq40uwIDAQABAoIBACclmiUZyyGomatl
      xXWGxIQazeZaiFQQut4aq03+LxUg16v3IWPAN8bT0jC94hj2HYC6Yh7zd/S5u3wT
      UDjGfDd9hO1XCTK2LH8vMng6k4uD7lyjU2m1+XdQTfEio1jNsQX7eDIuTNvMUuQH
      b/b52NFfWbfeNkmmlwaV9+YpIsy13++KJzIn/NlCHkOiMvtHTvf3ZUsPr9Kr6STJ
      7tt0+V9qPcajLckMMKqUoKgNcYAXkWgtM2BQteewPDSNXtxcviBVxHmsfyjuK0bN
      4rXOLhDua4bkbb6FslJXYpyiz7/cLqJygmGRDW9mf+I0QSYFstwQRzxchcS57upJ
      5a18GLECgYEA6oimH0I2b3rghSpS84CuEJcTjyUXvf1D5HeI5vHEvo0l9K1/E+of
      SLz3w6qsUBOFdSF0Vm/k5S75Sska6EGhtIe+agbhCNdULRjBvz5LToIDqovBcGq3
      sBnJxW4LQ2PQM68OXN2Wl4UEWF9Umis5CZsl+i81GiAVnnNjgh8XYkkCgYEAx00+
      KorTQBe5c9/uYPvYEHG7FsVSprMWsm3GWOJNhqgW27EkKqGRPLu2UylOeYe0eyj3
      UkKFZHxl6Kt+xxAH50lcZ3aT6wUf9cXWVLcOy7IR+5FAkqeGvgKexbNZkLKiArfe
      SQo5FF7GIu0gP8W/fgv0r5itrAQwxOBnTQQBnuMCgYEAjHRpiC7PCtQ7wYQnSUy2
      8ZiITiGYpl8WWax8gFIp0TQWlwGQKQz8z0Lb3oJHz2zhb9QpJ9q66cXH5dGqG42y
      mbrxfe3Attq9voQlA7L6xnl2WJx5rCk8+Gl5PJM6i5ErDsi3gUXy+arff00YDXv1
      HJudksbStmKgj9Pqs/KKvoECgYAWmePa3zNlqUsWoOZfiS/PbZZR1r6wuM5yHZDI
      s6EnDBjLgSMg0oGt6XuboquLjKAi91pUscZ+xrynzgrqeB7tU5xu/zt3A3XEYVMU
      +E1tPBxd8vLnrqfRFGr88IHPrvJAbKmAjvA6JyVBALMPiFVW7fQplZ7cSv1c1jXg
      vfuREQKBgE77qPlhDBwRqMB8H4A7Ai8lj1OrYbnqau4hGto3CUa5c8HKTx8LBvod
      jb3PUVkXrXmITZi77LH/wpV+ktuxmox+XRbfNZRZjACz2sts+She3A+nVa+jTPAa
      BMgC7znkT50NXccDayxmS62oqLbnOaGxKpk4h8UfKv5qh+NbEfiM
      -----END RSA PRIVATE KEY-----
- path: /run/kubeadm/kubeadm.yaml
  owner: root:root
  permissions: '0640'
  content: |
    ---
    apiServer: {}
    apiVersion: kubeadm.k8s.io/v1beta3
    clusterName: test
    controlPlaneEndpoint: 127.0.0.1:6443
    controllerManager: {}
    dns: {}
    etcd: {}
    kind: ClusterConfiguration
    kubernetesVersion: v1.26.6
    networking:
      dnsDomain: cluster.local
      podSubnet: 192.168.0.0/16
    scheduler: {}
    
    ---
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: InitConfiguration
    localAPIEndpoint: {}
    nodeRegistration:
      imagePullPolicy: IfNotPresent
      taints: null
- path: /run/cluster-api/placeholder
  owner: root:root
  permissions: '0640'
  content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
`))
	})
})
