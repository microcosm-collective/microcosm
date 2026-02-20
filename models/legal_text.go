package models

import (
	"text/template"
	"time"
)

var (
	legalServiceAgreement = template.Must(template.New("serviceAgreement").Parse(GetRawTermsOfServiceText()))
	legalTermsOfUse       = template.Must(template.New("termsOfUse").Parse(GetRawTermsOfUseText()))
	legalPrivacyPolicy    = template.Must(template.New("privacyPolicy").Parse(GetRawPrivacyPolicyText()))
	legalCookiePolicy     = template.Must(template.New("cookiePolicy").Parse(GetRawCookiePolicyText()))
)

var legalServiceAgreementLastModified, _ = time.Parse(time.DateOnly, "2014-03-06")

// GetRawTermsOfServiceText returns the terms of service
func GetRawTermsOfServiceText() string {
	const doc string = `<h2>Terms and Conditions for the Provision of Forum Services to Administrators</h2>

<h2 id="1">1. Introduction</h2>

<p id="1.1">1.1. These Terms of Service (together with the documents referred to herein) tell you the terms and conditions on which you may make use of our Service as a company, business, organisation or person.</p>

<p id="1.2">1.2. Please read these Terms of Service carefully before you sign up to our Service. By signing up to our Service and clicking the acceptance box, you agree to these Terms of Service on behalf of yourself or the company, business or organisation you represent.</p>

<h3 id="2">2. Defined terms</h3>

<p>In this Agreement, save where the context requires otherwise, the following words and expressions have the following meaning:</p>

<dl>
<dt>"<strong>Account</strong>"</dt>
<dd>means your registered account with us to access and use the Service;</dd>
</dl>

<dl>
<dt>"<strong>Administrator</strong>"</dt>
<dd>means any person you authorise to access your Account and who (i) manages your Account and/or (ii) exercises administrative control of the forum through your Account;</dd>
</dl>

<dl>
<dt>"<strong>Agreement</strong>"</dt>
<dd>means the agreement between {{.LegalEntity}} and you on these Terms of Service and the terms in the Paid Plan you choose;</dd>
</dl>

<dl>
<dt>"<strong>API</strong>"</dt>
<dd>means our application programming interface permitting you to access Forum Data on the Service;</dd>
</dl>

<dl>
<dt>"<strong>Confidential Information</strong>"</dt>
<dd>means information which is identified as confidential or proprietary by either party or the nature of which is clearly confidential or proprietary;</dd>
</dl>

<dl>
<dt>"<strong>Forum Data</strong>"</dt>
<dd>means the data we host on your behalf as a result of your use of the Service including (without limitation) any data inputted by you, your Administrator and Forum Users;</dd>
</dl>

<dl>
<dt>"<strong>Fees</strong>"</dt>
<dd>means the fees we charge for your use of the Service as set out in the Paid Plan as available here <a href="{{.LinkToFees}}">{{.LinkToFees}}</a>;</dd>
</dl>

<dl>
<dt>"<strong>Forum User</strong>"</dt>
<dd>means any individual other than the Administrator that you permit to view and/or contribute content to the forum.</dd>
</dl>

<dl>
<dt>"<strong>Intellectual Property Rights</strong>"</dt>
<dd>means (a) patents, trademarks, service marks, registered designs, applications for any of those rights, trade and business names (including Internet domain names), unregistered trademarks, unregistered trade and business names, database rights, copyrights, rights in designs and inventions; and (b) rights of the same or similar effect to those specified in paragraph (a); in each case, in any jurisdiction;</dd>
</dl>

<dl>
<dt>"<strong>{{.LegalEntity}}</strong>"</dt>
<dt>"<strong>We</strong>"</dt>
<dt>"<strong>Us</strong>"</dt>
<dd>means {{.LegalEntity}}, having its registered office at {{.MicrocosmAddress}};</dd>
</dl>

<dl>
<dt>"<strong>Month</strong>"</dt>
<dd>means a monthly period commencing from the Start Date;</dd>
</dl>

<dl>
<dt>"<strong>Paid Plan</strong>"</dt>
<dd>means the paid plan you select when signing up to the Service which sets out the Fees we charge for your use of the Service as available here <a href="{{.LinkToFees}}">{{.LinkToFees}}</a>;</dd>
</dl>

<dl>
<dt>"<strong>Service</strong>"</dt>
<dd>means the online forum software and forum hosting we make available as a service and the API;</dd>
</dl>

<dl>
<dt>"<strong>Subscription Period</strong>"</dt>
<dd>means the period of time we agree to supply the Services in accordance with your Paid Plan.</dd>
</dl>

<dl>
<dt>"<strong>Start Date</strong>"</dt>
<dd>means the date you sign up to the Service;</dd>
</dl>

<dl>
<dt>"<strong>You</strong>"</dt>
<dt>"<strong>Your</strong>"</dt>
<dd>means the person, company, business or any other entity or organisation signing up to the Service acting through the Administrator or other representative.</dd>
</dl>

<h3 id="3">3. The forum Service</h3>

<p id="3.1">3.1. {{.LegalEntity}} provides you with an efficient and user friendly way to develop and run a forum. You can use our Service to design and customise your forum to meet the requirements of your forum community.</p>

<p id="3.2">3.2. Our forums are designed to reflect the modern needs and expectations of users.  They support mobile viewing and provide functionality that users have come to expect from online social resources such as event planning, galleries, classifieds, reviews, articles, Q&amp;A and polls.</p>

<h3 id="4">4. Duration and Cancellation</h3>

<p id="4.1">4.1. This Agreement shall commence on the Start Date and shall continue unless terminated in accordance with clause 4.2 or clause 19.</p>

<p id="4.2">4.2. You may cancel your Account at any time as indicated in your Account settings for the Service and any cancellation will take effect at the end of the current Subscription Period.</p>

<p id="4.3">4.3. Your access and the access of Administrators and Forum Users will cease when your cancellation of the Service takes effect.</p>

<h3 id="5">5. Access to the Service</h3>

<p id="5.1">5.1. We grant you a non-exclusive, non-transferable and non sub-licensable licence to permit your Administrator and Forum Users to access and use the Service as permitted by the functionality of the Service and your Account.</p>

<p id="5.2">5.2. To access the Service, your Administrator will need to create an Account using a valid email address and password.</p>

<p id="5.3">5.3. You will ensure that your Administrator shall keep secure and confidential any passwords for his or her use of the Service.</p>

<p id="5.4">5.4. If we discover that any Administrator is sharing their email address and passwords for the Service with any person who you have not authorised to act as an Administrator, we may suspend your Account and/or terminate this Agreement immediately.</p>

<p id="5.5">5.5. You are responsible for maintaining the confidentiality of your email addresses and passwords and any activities that occur under your Account. Whenever applicable, we encourage you to not reuse passwords and to use “strong” passwords (passwords that use a combination of upper and lower case letters, numbers and symbols) with your Account. If you have any concerns about your Account you should contact us at <a href="mailto:{{.MicrocosmEmail}}">{{.MicrocosmEmail}}</a>.</p>

<p id="5.6">5.6. We permit you to use the API (including updates to the API that we may make available from time to time) for the sole purpose of developing services and applications that permit Forum Users to: (i) access Forum Data; and/or (ii) contribute content to the forum.  You are not permitted to use the API for developing services and applications to exercise the administrative controls of an Administrator.</p>

<p id="5.7">5.7. You shall use the API in accordance with the reasonable directions given by us from time to time.</p>

<p id="5.8">5.8. You must use the most current version of the API after any previous version has been upgraded, following notice from us to do so.</p>

<p id="5.9">5.9. We may monitor your use of the Service to ensure quality, improve the Service, and verify your compliance with the Agreement.</p>

<h3 id="6">6. Restrictions on use of the service</h3>

<p id="6.1">6.1. The Service is only for the use by your Administrator and Forum Users as permitted by the functionality of the Service.</p>

<p id="6.2">6.2. You must comply with any conditions or restrictions contained in your Paid Plan.</p>

<p id="6.3">6.3. When using the forum you must not and you will procure that your Administrator or Forum Users must not:</p>

<ol type="i">

<li>circumvent, disable or otherwise interfere with any security related features of the Service or features that prevent or restrict use or copying of the content accessible via the Service;</li>

<li>give any false information in your account details;</li>

<li>give false information in a Forum User's account;</li>

<li>permit a person to use your Account who has not been authorised by you to do so;</li>

<li>use the Service if we have suspended or banned you or them from using it;</li>

<li>send junk, spam or repetitive messages;</li>

<li>engage in any illegal or unlawful conduct;</li>

<li>modify, interfere, intercept, disrupt or hack the Service;</li>

<li>misuse the Service by knowingly introducing viruses, Trojans, worms, logic bombs or other similar material or by undertaking any other action which would harm the Service or any Service user's equipment or have a detrimental impact on any user's experience of using the Service;</li>

<li>collect any data from the Service other than in accordance with these Terms of Service;</li>

<li>submit or contribute to the Service any content that is abusive, threatening, obscene, untrue, shocking or offensive;</li>

<li>abuse, harm or bully another user, member of our staff or person through the Service;</li>

<li>submit or contribute any content to the Service without the permission of the content owner or otherwise infringe the copyright, trademark or other rights of third parties; or</li>

<li>submit or contribute any information or commentary about another person which is untrue, private, confidential or defamatory (including personally accusing another person of unproven criminality or serious wrongdoing which could damage their reputation in the eyes of anyone reading the comment).</li>
</ol>

<p id="6.4">6.4. Failure to comply with clause 6.3 constitutes a serious breach of these Terms of Service and may result in our taking all or any of the following actions (with or without notice):</p>

<ol type="a">
<li>immediate, temporary or permanent withdrawal of your right to use the Service;</li>
<li>immediate, temporary or permanent removal of any Forum Data;
issuing of a warning to you;</li>
<li>legal action against you including proceedings for reimbursement of all costs (including, but not limited to, reasonable administrative and legal costs) resulting from the breach; and/or</li>
<li>disclosure of such information to law enforcement authorities as we reasonably feel is necessary.</li>
</ol>

<p id="6.5">6.5. The responses described in clause 6.4 are not limited, and we may take any other action we reasonably deem appropriate.</p>

<p id="6.6">6.6. We may, without liability to you, disable the Service to the extent necessary to disable access to any data or material that breaches the provisions of clause 6.</p>

<p id="6.7">6.7. You shall not, except as may be allowed by any applicable law which is incapable of exclusion by us and except to the extent expressly permitted under this Agreement:</p>

<ol type="a">
<li>attempt to copy, modify, duplicate, create derivative works from, frame, mirror, republish, download, display, transmit, or distribute all or any portion of the Service in any form or media or by any means;</li>

<li>attempt to reverse compile, disassemble, reverse engineer or otherwise reduce to human-perceivable form all or any part of the Service;</li>

<li>access all or any part of the Service in order to build a product or service which competes with the Service;</li>

<li>use the Service to provide services to third parties other than Forum Users;</li>

<li>license, sell, rent, lease, transfer, assign, distribute, display, disclose, or otherwise commercially exploit, or otherwise make the Service available to any third party except the Administrator and Forum Users as provided for in this clause 6;</li>

<li>attempt to obtain, or assist third parties in obtaining access to the Service other than as provided under this clause 6.</li>

</ol>

<p id="6.8">6.8. You shall use all reasonable endeavours to prevent any unauthorised access to, or use of, the Service and, in the event of any such unauthorised access or use, promptly notify us.</p>

<p id="6.9">6.9. Parts of the Service are provided using open source software. The open source software we use is detailed on our development pages <a href="{{.LinkToOpenSourceLicenses}}">{{.LinkToOpenSourceLicenses}}</a> and may be used in accordance with the licence for that software.</p>

<h3 id="7">7. Hyperlinks</h3>

<p id="7.1">7.1. We may amend, re-format, modify or replace any website link added to the forum. We may do this to:</p>

<ol type="i">
<li>improve the appearance of the link;</li>
<li>ensure the link uses one of our partners or affiliates;</li>
<li>direct Forum Users to an interstitial or warning page when we suspect the link is to something that may harm Forum Users or their devices such as links to malware or a shock site; and</li>
<li>change how the link is displayed so that information from the link destination (such as content and media from the linked webpage) is displayed within the forum.  For example, we may embed video or other media from a linked webpage into the forum.</li>
</ol>

<h3 id="8">8. Verification Service</h3>

<p id="8.1">8.1 We will provide a Forum User verification service to your Forum Users through the Service.  Our verification service indicates we have been able to verify the identity of the user in accordance with our verification policy <a href="{{.LinkToVerificationPolicy}}">{{.LinkToVerificationPolicy}}</a>.</p>

<p id="8.2">8.2. A user verification is not: (i) an endorsement by us of the user or their statements and representations; (ii) any guarantee or promise in relation to fitness, quality or description of any goods or services the user sells or promotes; or (iii) a representation by us that the user is trustworthy, reliable or honest.</p>

<h3 id="9">9. Forum legal documents</h3>

<p id="9.1">9.1. When you create your forum we will provide you the option of linking to: (i) your own terms of use, privacy policy and cookie policy (or other legal documents); or (ii) a precedent standard form terms of use, privacy policy and cookie policy that we provide.</p>

<p id="9.2">9.2. If you choose to use our precedent standard form terms of use, privacy policy and cookie policy you do so at your own risk and we advise you to seek your own legal advice in relation to their suitability for your forum and use.</p>

<p id="9.3">9.3. You may amend our precedent standard form terms of use, privacy policy and cookie policy as you believe is necessary.</p>

<p id="9.4">9.4. If you use your own terms of use, privacy policy and cookie policy or amend our precedent standard form terms of use, privacy policy and cookie policy you will ensure that the terms of those agreements substantially reflect your obligations to us under this Agreement.</p>

<h3 id="10">10. Service availability and support</h3>

<p id="10.1">10.1. We will use commercially reasonable endeavours to make the Service available except for:</p>

<ol type="a">
<li>planned maintenance for which 24 hours' notice will be given; or</li>
<li>unscheduled maintenance during normal business hours (UK time) or otherwise, for which we will use reasonable endeavours to give you advance notice.</li>
</ol>

<p id="10.2">10.2. We will, as part of the Service, use reasonable endeavours to provide a level of support that is appropriate to the nature of any issues requiring support during normal business hours (UK time).</p>

<h3 id="11">11. Your obligations</h3>

<p id="11.1">11.1. You shall provide us with all necessary cooperation in relation to this Agreement and access to such information as may be required by us.</p>

<p id="11.2">11.2. You will:</p>

<ol type="a">

<li>comply with all applicable laws and regulations with respect to your activities under this Agreement;</li>

<li>carry out all your responsibilities and obligations in a timely and efficient manner. In the event of any delays in the provision of such assistance or performance of any obligations as agreed by the parties, we may adjust any agreed timetable or delivery schedule as reasonably necessary;</li>

<li>ensure that the Administrator and Forum Users use the Services in accordance with the terms and conditions of this Agreement and shall be responsible for any Administrator or Forum User's actions and omissions in connection with their use of the Service;</li>

<li>obtain and shall maintain all necessary licences, consents, and permissions necessary for us or our contractors and agents to perform their obligations under this Agreement including without limitation the Services;</li>

<li>ensure that your network and systems, including the device used by each Administrator to access the Service, comply with the relevant specifications provided by us from time to time; and</li>

<li>review our best practice guidelines <a href="{{.LinkToBestPractises}}">{{.LinkToBestPractises}}</a>. (Our best practice guidelines do not form part of this Agreement but they do contain a useful overview of your obligations under this Agreement and some guidance on how to avoid liability for the content Forum Users add to your forum).</li>
</ol>

<h3 id="12">12. Forum data</h3>

<p id="12.1">12.1. We do not claim ownership in your Forum Data. You retain any right, title and interest in and to the Forum Data although you grant us a licence to use the Forum Data solely for the purpose of providing the Service. You shall have sole responsibility for the legality, reliability, integrity, accuracy and quality of the Forum Data.</p>

<p id="12.2">12.2. We shall not be responsible or liable for the deletion, correction, alteration, destruction, damage, loss, disclosure or failure to store any Forum Data using the Service.</p>

<p id="12.3">12.3. If we process any Personal Data as a result of hosting the Forum Data or as a result of your use of the Service, you agree that we do so as Data Processor and that you are the Data Controller in relation to such Personal Data.</p>

<p id="12.4">12.4. We agree that, in relation to such Personal Data:</p>

<ol type="a">
<li>we will process the Personal Data only in accordance with the terms of this Agreement and any lawful written instructions reasonably given by you from time to time;</li>

<li>you shall ensure that you are entitled to transfer the relevant Personal Data to us so that we may lawfully use, process and transfer the personal data in accordance with this Agreement on your behalf; and</li>

<li>we will both have in place appropriate technical and organisational security measures against unauthorised or unlawful processing of Personal Data and against accidental loss or destruction of, or damage to, Personal Data.</li>
</ol>

<p id="12.5">12.5. Each party shall comply with its relevant obligations under all applicable data protection legislation and shall ensure that it has obtained all consents as required by applicable data protection legislation as necessary for the purposes of this Agreement.</p>

<p id="12.6">12.6. For the purposes of this clause 12, the terms "Personal Data", "Data Processor" and "Data Controller" shall have the same meaning as set out in the UK Data Protection Act 1998.</p>

<h3 id="13">13. Charges and payment</h3>

<p id="13.1">13.1. Our Fees are payable commencing from the Start Date as they become due in accordance with your Paid Plan.</p>

<p id="13.2">13.2. You will provide us with valid, up-to-date and complete payment details on signing up to the Service and you hereby authorise us to charge your chosen payment method for the Fees in accordance with your Paid Plan. You are responsible for keeping your payment method details up-to-date.</p>

<p id="13.3">13.3. We will charge your chosen payment method in accordance with your Paid Plan.</p>

<p id="13.4">13.4. If we are unsuccessful in charging your chosen payment method and have still not received payment within 14 days after informing you and without prejudice to any other rights and remedies available to us we may, without liability to you, suspend or temporarily disable all or part of your access to the Services and we shall be under no obligation to provide any or all of the Services while the Fees concerned remain unpaid.</p>

<p id="13.5">13.5. All amounts and Fees stated or referred to in this Agreement:</p>

<ol type="a">
<li>are payable in the currency specified in the Paid Plan or otherwise stipulated by us;</li>

<li>are exclusive of value added tax (if applicable) unless otherwise expressly stated which shall be paid at the same time as payment of the Fees. We shall send you a VAT invoice if you request us to do so.</li>
</ol>

<p id="13.6">13.6. Unless we agree otherwise in writing, we may increase the Fees upon 30 days' notice in writing to you. Any increase in the Fees will not take effect until the end of your current Subscription Period. If you are unhappy with the increase, you may terminate this Agreement under clause 4.2.</p>

<h3 id="14">14. Proprietary rights</h3>

<p>You acknowledge and agree that we and/or our licensors own all intellectual property rights in the Service. Except as expressly stated herein, this Agreement does not grant you any rights to, or in, patents, copyrights, database rights, trade secrets, trade names, trademarks (whether registered or unregistered), or any other rights or licences in respect of the Service.</p>

<h3 id="15">15 Confidentiality</h3>

<p id="15.1">15.1. Each party may be given access to Confidential Information from the other party in order to perform its obligations under these terms and conditions. A party's Confidential Information shall not be deemed to include information that:</p>

<ol type="a">
<li>is or becomes publicly known other than through any act or omission of the receiving party;</li>

<li>was in the other party's lawful possession before the disclosure;</li>

<li>is lawfully disclosed to the receiving party by a third party without restriction on disclosure;</li>

<li>is independently developed by the receiving party, which independent development can be shown by written evidence; or</li>

<li>is required to be disclosed by law, by any court of competent jurisdiction or by any regulatory or administrative body.</li>
</ol>

<p id="15.2">15.2. Each party shall hold the other's Confidential Information in confidence and, unless required by law, not make the other's Confidential Information available for use for any purpose other than as needed to perform this Agreement.</p>

<p id="15.3">15.3. Each party shall take all reasonable steps to ensure that the other's Confidential Information to which it has access is not disclosed or distributed by its employees or agents in violation of these terms and conditions.</p>

<p id="15.4">15.4. Each party shall take a back-up of its own Confidential Information and shall not be responsible to the other for any loss, destruction, alteration or disclosure of Confidential Information.</p>

<p id="15.5">15.5. We acknowledge that the Forum Data which is not publicly available through the Forum is your Confidential Information.</p>

<p id="15.6">15.6. This clause 15 shall survive termination of this Agreement, however arising.</p>

<h3 id="16">16. Limited Warranty</h3>

<p id="16.1">16.1. We undertake to make the Service available as specified in clause 10.1. Otherwise, the Service is provided on an “as is” basis and we give no representations, conditions, warranties or other terms of any kind in respect of the Service.</p>

<p id="16.2">16.2. Except as expressly and specifically provided in this Agreement:</p>

<ol type="a">
<li>all warranties, representations, conditions and all other terms of any kind whatsoever implied by statute or common law are, to the fullest extent permitted by applicable law, excluded from this Agreement; and</li>

<li>we will not be responsible for any interruptions, delays, failures or non-availability affecting the Service or performance of the Service which are caused by third party services or errors or bugs in software, hardware or the Internet on which the Service rely as you acknowledge that we do not control such third party services and such errors or bugs are inherent in the use of such software, hardware and the Internet.</li>
</ol>

<h3 id="17">17. Liability</h3>

<p id="17.1">17.1. This clause 17 sets out our entire financial liability (including any liability for the acts or omissions of our employees, agents and sub-contractors) to you in respect of:</p>

<ol type="a">
<li>any breach of this Agreement;</li>

<li>any use made by you or your Administrator or Forum Users of the Service or any part of it; and</li>

<li>any representation, statement or tortious act or omission (including negligence) arising under or in connection with this Agreement.</li>

</ol>

<p id="17.2">17.2. Nothing in this Agreement excludes our liability:</p>

<ol type="a">
<li>for death or personal injury caused by our negligence; or</li>

<li>for fraud or fraudulent misrepresentation.</li>
</ol>

<p id="17.3">17.3. We shall not be liable whether in tort (including for negligence or breach of statutory duty), contract, misrepresentation, restitution or otherwise for any loss of profits, loss of business, depletion of goodwill and/or similar losses or loss or corruption of data or information, or pure economic loss, or for any special, indirect or consequential loss, costs, damages, charges or expenses however arising under this Agreement.</p>

<p id="17.4">17.4. To the extent that we are liable under this Agreement, our total aggregate liability in contract, tort (including negligence or breach of statutory duty), misrepresentation, restitution or otherwise, arising in connection with the performance or contemplated performance of this Agreement shall be limited to the total Fees paid in the twelve (12) month period immediately preceding the date on which the liability arose.</p>

<h3 id="18">18. Indemnity</h3>

<p>You shall indemnify and defend us and our agents and contractors from and against any and all losses, damages, claims, liabilities or expenses (including reasonable attorneys' fees) arising out of a claim brought by any third party (including Forum Users) relating to your use of the Service (except to the extent caused by our negligence) including without limitation our hosting and use of Forum Data and your decisions, financial, business and commercial judgments, and acts or omissions, based on your use of the Service.</p>

<h3 id="19">19. Termination</h3>

<p id="19.1">19.1. We may terminate this Agreement by notice to you with immediate effect or such notice as we may elect to give you, if you:</p>

<ol type="a">
<li>are in breach of applicable law;</li>

<li>infringe our intellectual property rights in the Service or make unauthorised use of our Confidential Information; or</li>

<li>fail to pay any sum due to us and such sum has remained outstanding for fourteen (14) days following notice requiring such sum to be paid.</li>
</ol>

<p id="19.2">19.2. Either party may terminate this Agreement at any time on written notice to the other if the other:</p>

<ol type="a">
<li>is in material or persistent breach of any of the terms of this Agreement and either that breach is incapable of remedy, or the other party fails to remedy that breach within 30 days after receiving written notice requiring it to remedy that breach; or</li>

<li>is unable to pay its debts (within the meaning of section 123 of the Insolvency Act 1986), or becomes insolvent, or is subject to an order or a resolution for its liquidation, administration, winding-up or dissolution (otherwise than for the purposes of a solvent amalgamation or reconstruction), or has an administrative or other receiver, manager, trustee, liquidator, administrator or similar officer appointed over all or any substantial part of its assets, or enters into or proposes any composition or arrangement with its creditors generally, or is subject to any analogous event or proceeding in any applicable jurisdiction.</li>
</ol>

<p id="19.3">19.3. You may terminate this Agreement at anytime by giving us 30 days notice to terminate this Agreement. Such termination will be effective from the end of the Subscription Period in which the final day of the notice period occurs in.</p>

<p id="19.4">19.4. On termination of this Agreement for any reason all licences granted under this Agreement shall immediately terminate and your right to access and use the Service will end.</p>

<p id="19.5">19.5. The accrued rights of the parties as at termination, or the continuation after termination of any provision expressly stated to survive or implicitly surviving termination shall not be affected or prejudiced.</p>

<h3 id="20">20. Transfer of the Forum to a New Operator</h3>

<p id="20.1">20.1. If this Agreement is terminated in accordance with clause 19.1, clause 19.2 or clause 19.3, you agree that we may transfer the Service and associated Forum Data (excluding your non-public Confidential Information and intellectual property rights but including all Forum Data licensed from Forum Users and anything else that comprises the forum) to a Forum User elected by the other Forum Users to take over the operation of the forum.</p>

<p id="20.2">20.2. If you object to any transfer pursuant to clause 20.1 you must notify us within 14 days of receipt of notice from us that we intend to exercise our right to transfer the Service and associated Forum Data. No transfer will take place under clause 20.1 if you object to the transfer in accordance with this clause 20.2.</p>

<p id="20.3">20.3. Our right to transfer the Service and associated Forum Data under clause 20.1 will continue for 30 days from the date of notice to terminate this Agreement in accordance with clause 19.1, clause 19.2 or clause 19.3.</p>

<p id="20.4">20.4. You agree that following notice to terminate this Agreement in accordance with clause 19.1, clause 19.2 or clause 19.3, a poll will be added to the forum by us for the election of a new forum operator. The poll will be accompanied and may be subsequently updated with information from us to Forum Users about the requirement for a new forum operator and the transfer process. No information will be provided to Forum Users about the reasons for the transfer unless you choose to make such information public.  The poll will run for a reasonable time period decided at our discretion in order for a Forum User to be selected to run the forum. We are permitted to discuss the reasons for the transfer with a Forum User elected in accordance with clause 20.5 provided the Forum User is subject to obligations of confidentiality.</p>

<p id="20.5">20.5. Forum Users must nominate themselves to take part in the poll to select a new operator for the forum. We may offer to transfer the Service and associated Forum Data to the Forum User with the highest number of votes from other Forum Users in the poll.  If the Forum User declines to take over the operation of the forum, we may offer to transfer the Service and associated Forum Data to other Forum Users that receive votes in the poll in order of their standing in the final poll (highest to lowest). If none of the nominated Forum Users agree to the transfer of the forum, the forum will be closed.  The forum will remain open for Forum Users to access during the period of the election and while the subsequent transfer takes place.</p>

<p id="20.6">20.6. Following any written notice from you that we receive either before or after the transfer of the Service and associated Forum Data in accordance with clause 20.1, we will remove any trademark of yours that you disclosed publicly on the forum. To exercise your right under this clause 20.4 you must notify us of the trademark you wish to have removed and its location.</p>

<p id="20.7">20.7. You agree to cooperate in good faith with us and any Forum User for a transfer of the forum in accordance with this clause 20, including you shall promptly execute and deliver such documents and perform such acts as may reasonably be required for the purpose of giving full effect to the transfer of Service and the associated Forum Data.</p>

<h3 id="21">21. Force majeure</h3>

<p>No party shall be liable to the other for any delay or non-performance of its obligations under this Agreement arising from any cause beyond its control including, without limitation, any of the following: non-availability or failure of any third party service, act of God, governmental act, war, fire, flood, explosion, communications or failure of the Internet or civil commotion.</p>

<h3 id="22">22. Waiver</h3>

<p>No forbearance or delay by either party in enforcing its rights shall prejudice or restrict the rights of that party and no waiver of any such rights or of any breach of any contractual terms shall be deemed to be a waiver of any other right or of any later breach.</p>

<h3 id="23">23. Severability</h3>

<p>If any provision of this Agreement is judged to be illegal or unenforceable, the continuation in full force and effect of the remainder of the provisions shall not be prejudiced.</p>

<h3 id="24">24. Amendments</h3>

<p>We may amend this Agreement at any time. We will post a copy of the amended Terms of Service to our website <a href="http://microcosm.app">http://microcosm.app</a>. Your continued use of the Service after this Agreement has been amended will be considered as your acceptance of the amended Agreement. We will notify you of any material changes to this Agreement and give you at least 30 days' notice.</p>

<h3 id="25">25. Assignment</h3>

<p>You may not assign or purport to assign this Agreement or any of its obligations there under without our prior written consent.</p>

<h3 id="26">26. No partnership or agency</h3>

<p>Nothing in this Agreement is intended to or shall operate to create a partnership between the parties, or authorise either party to act as agent for the other, and neither party shall have the authority to act in the name or on behalf of or otherwise to bind the other in any way (including, but not limited to, the making of any representation or warranty, the assumption of any obligation or liability and the exercise of any right or power).</p>

<h3 id="27">27. Third party rights</h3>

<p>This Agreement does not confer any rights on any person or party (other than the parties to this Agreement and, where applicable, their successors and permitted assigns) pursuant to the Contracts (Rights of Third Parties) Act 1999.</p>

<h3 id="28">28. Notices</h3>

<p>Any notice required to be given pursuant to this Agreement shall be given by email to <a href="mailto:{{.MicrocosmEmail}}">{{.MicrocosmEmail}}</a> in respect of notices to us and to the Administrator using the email address provided on signing up to the Service in respect of notices to you. Notices sent by email shall be deemed to have been received instantaneously on transmission, provided that they are sent to correct email address.</p>

<h3 id="29">29. Entire agreement</h3>

<p>This Agreement contains the whole agreement between the parties relating to the subject matter hereof and supersedes all prior agreements, arrangements and understandings between the parties relating to that subject matter.</p>

<h3 id="30">30. Governing law and jurisdiction</h3>

<p>This Agreement shall be governed by and construed in accordance with English law and each party hereby submits to the exclusive jurisdiction of the English courts.</p>

<p>Last modified {{.ServiceAgreementLastModified}}</p>
`
	return doc
}

var legalTermsOfUseLastModified, _ = time.Parse(time.DateOnly, "2026-02-20")

// GetRawTermsOfUseText returns the terms of use
func GetRawTermsOfUseText() string {
	const doc string = `<h2>{{.CustomerName}}</h2>

<h2>Forum Terms of Use</h2>

<h3>Welcome to our forum!</h3>

<p>The forum <a href="{{.CustomerURL}}">{{.CustomerURL}}</a> (together with all sub-domains on {{.CustomerURL}}, the "<strong>forum</strong>") is operated by The Microcosm Collective CIC, a community interest company incorporated under the laws of England and Wales with company number 16265268 ("<strong>we</strong>" or "<strong>us</strong>").</p>

<p>If you need to contact us, please use our contact details set out at the end of this document.</p>

<h3>Your relationship with us</h3>

<p>This document and any documents referred to within it (collectively, and as may be updated from time to time, the "<strong>Terms of Use</strong>") set out the terms that govern your use of the forum. It is important that you read and understand the Terms of Use before using the forum. If there is anything within them that you do not understand, then please contact us to discuss what this means for you.</p>

<p>By setting up an account to use the forum, or using and accessing the forum, you agree to these Terms of Use, the most updated version of which can be located at <a href="https://www.lfgss.com/about/terms/">https://www.lfgss.com/about/terms/</a>. If you do not agree to these Terms of Use, please do not browse or otherwise access or use the forum.</p>

<h3>Online safety act</h3>

<p>In order to comply with the Online Safety Act 2023 (UK), you are expected to report any Illegal Content which you may come across on the forum to moderators. Illegal Content is defined in section 59 of the Online Safety Act 2023, but by way of high level summary, it includes content which relates to 17 types of priority offences and 5 types of non-priority offences which are listed below.</p>

<p><strong>Priority</strong>: Content relating to</p>

<ul>
	<li>Terrorism offences</li>
	<li>Child sexual exploitation and abuse, and child sexual abuse material</li>
	<li>Grooming</li>
	<li>Hate offences under the law</li>
	<li>Harassment, stalking threats and abuse which is likely to:
		<ul>
			<li>cause fear or alarm,</li>
			<li>stir up racial hatred,</li>
			<li>stir up hatred on the basis of religion or sexual orientation, or</li>
			<li>may provoke violence;</li>
		</ul>
		or threatening behaviour which is likely to cause harassment or distress
	</li>
	<li>Controlling or coercive behaviour</li>
	<li>Intimate image abuse</li>
	<li>Extreme pornography</li>
	<li>Sexual exploitation of adults</li>
	<li>Human trafficking</li>
	<li>Unlawful immigration</li>
	<li>Fraud and financial services</li>
	<li>Proceeds of crime, including in particular the re-sale of stolen bicycles or parts</li>
	<li>The unlawful supplying of drugs and psychoactive substances</li>
	<li>Offences in relation to firearms and other weapons</li>
	<li>Encouraging suicide or assisting suicide</li>
	<li>Foreign Interference (under s13 of the National Security Act 2023)</li>
	<li>Animal welfare offences</li>
</ul>

<p><strong>Non-priority</strong>: Content relating to</p>

<ul>
	<li>Epilepsy trolling</li>
	<li>Cyberflashing with the intent of causing alarm, distress or humiliation in the receiver, or for the purpose of sexual gratification on the behalf of the sender</li>
	<li>Encouraging self-harm</li>
	<li>False communications with the intent of causing non-trivial psychological or physical harm to the likely audience</li>
	<li>Obscene content relating to torture or extreme cruelty</li>
</ul>

<p></p>

<p>Please refer to section 59 of the Online Safety Act 2023 and associated guidance issued by Ofcom if you are unclear as to whether content may be illegal content and should be reported, but we would encourage you to err on the side of caution when reporting posts for illegal content. A slightly more detailed description can also be found in the Appendix to these Terms of Use. Please note that content which is not Illegal Content may still violate these Terms of Use, and all users should comply with the requirements of the Online Safety Act 2023 as well as the rest of these Terms of Use.</p>

<h3>Information about you</h3>

<p>Your privacy is important to us. You should read the forum privacy policy <a href="{{.LinkToPrivacyPolicy}}">{{.LinkToPrivacyPolicy}}</a> to understand how we collect, use and share information about you.</p>

<h3>Setting up an account</h3>

<p>You can view some areas of the forum without creating an account. However, to leave comments or access certain restricted areas of the forum you must register with us and set up an account with an email address.</p>

<p>You are responsible for maintaining the confidentiality of your login details and any activities that occur under your account. If you have any concerns that your account may have been misused, you should contact us at <a href="mailto:support@microcosm.cc">support@microcosm.cc</a> straight away to let us know.</p>

<h3>Your right to use the forum</h3>

<p>We give you permission to use the forum provided your use complies with these Terms of Use.</p>

<p>You must be 18 years or older and capable in your country of residence of entering into a legal binding agreement to use our forum. Please inform a moderator immediately if you are aware of any person using the forum who may be under the age of 18.</p>

<p>Your right to use the forum is personal to you and you are not allowed to give this right to another person or to sell, gift or transfer your account to another person. You may imprint, copy, download or temporarily store extracts from the forum onto your computer or other device for the purpose of using the forum. Your right to use the forum does not stop us from giving other people the right to use the forum.</p>

<p>In legal terms the paragraphs above mean that we give you a personal, worldwide, royalty-free, non-assignable and non-exclusive licence to use the forum. This licence is for the sole purpose of enabling you to use and enjoy the benefits of the forum in a manner permitted by these Terms of Use.</p>

<h3>Our content</h3>

<p>The forum comprises content including pages, screens, information and other material and we have permission from the owner of the content to use it or it belongs to us. In legal terms, this means that we are the owner or the licensee of all intellectual property that comprises the forum and the collective work that comprises the data in the forum.</p>

<p>We do not claim ownership of any User Content you add to the forum (unless the content is owned by us) but you do give us permission to use the User Content you add to the forum in accordance with the section titled "Your content".</p>

<h3>Your content</h3>

<p>You confirm that images, sounds, text or information that you submit or create ("<strong>User Content</strong>") whilst using the forum will meet the Rules of Acceptable Use (defined in the section below).</p>

<p>You give us unlimited and irrevocable permission to use your User Content within the forum or for personal purposes, as long as such use is not in a commercial context, and allow other users of the forum to do the same for no fee. In legal terms this means that you grant us an irrevocable worldwide, non-exclusive, royalty-free and perpetual licence to use, copy, reproduce, distribute, adapt, re-format, modify, publish and translate the User Content in any form within the forum or for personal purposes, as long as such use is not in a commercial context.</p>

<p>Our right to use your User Content does not in any way affect your privacy rights or your right to use (including, for the avoidance of doubt, monetising) your User Content and we will only use information that identifies you as set out in the forum privacy policy <a href="{{.LinkToPrivacyPolicy}}">{{.LinkToPrivacyPolicy}}</a>.</p>

<p>We do not check or moderate any User Content before it is added to the forum by users. We may later check, moderate, reject, refuse or delete any User Content if anybody reports it to us and we believe that it breaks any of the Rules of Acceptable Use.</p>

<h3>Rules of Acceptable Use</h3>

<p>In addition to the other requirements within these Terms of Use, this section describes specific rules that apply to your use of the forum (the "<strong>Rules of Acceptable Use</strong>").</p>

<p>When using the forum you must not:</p>

<ul>
	<li>post, send or otherwise create any content on the forum which is Illegal Content (as defined in section 59 of the Online Safety Act 2023 and summarised above), or otherwise engage in any illegal or unlawful conduct</li>
	<li>sell or attempt to sell any property which you are aware has been, or suspect may have been, stolen</li>
	<li>circumvent, disable or otherwise interfere with any security related features of the forum or features that prevent or restrict use or copying of the content accessible via the forum;</li>
	<li>give any false information in your account details;</li>
	<li>take another person's identity without that person's permission or misrepresent you are acting on behalf of a person, entity or organisation;</li>
	<li>use the forum if we have suspended or banned you from using it;</li>
	<li>send junk, spam or repetitive messages;</li>
	<li>engage in any illegal or unlawful conduct,</li>
	<li>modify, interfere, intercept, disrupt or hack the forum;</li>
	<li>misuse the forum by knowingly introducing viruses, Trojans, worms, logic bombs or other similar material or by undertaking any other action which would harm the forum, any forum user's equipment or have a detrimental impact on any forum user's experience of using the forum;</li>
	<li>collect any data from the forum other than in accordance with these Terms of Use;</li>
	<li>submit or contribute any content that is abusive, threatening, obscene, untrue, shocking or offensive;</li>
	<li>abuse, harm or bully another forum user, member of our staff or person;</li>
	<li>submit or contribute any User Content without the permission of the content owner or otherwise infringe the copyright, trademark or other rights of third parties; or</li>
	<li>submit or contribute any information or commentary about another person which is untrue, private, confidential or defamatory (including personally accusing another person of unproven criminality or serious wrongdoing which could damage their reputation in the eyes of anyone reading your comment).</li>
</ul>

<p>Failure to comply with the Rules of Acceptable Use constitutes a serious breach of these Terms of Use and may result in our taking all or any of the following actions (with or without notice):</p>

<ul>
	<li>immediate, temporary or permanent withdrawal of your right to use the forum;</li>
	<li>immediate, temporary or permanent removal of any User Content;</li>
	<li>issuing of a warning to you;</li>
	<li>legal action against you including proceedings for reimbursement of all costs (including, but not limited to, reasonable administrative and legal costs) resulting from the breach; and/or</li>
	<li>disclosure of such information to law enforcement authorities as we reasonably feel is necessary.</li>
</ul>

<p>The responses described in this clause are not exhaustive or exclusive, and we may take any other action we reasonably deem appropriate.</p>

<h3>Notice and takedown policy</h3>

<p>If you believe that content available through the forum:</p>

<ul>
	<li>infringes your rights or any rights of a third party you represent;</li>
	<li>may be Illegal Content; or</li>
	<li>otherwise breaches the Rules of Acceptable Use,</li>
</ul>

<p>please tell us immediately by contacting a moderator on <a href="https://www.lfgss.com/conversations/172777/">https://www.lfgss.com/conversations/172777/</a>, or alternatively, sending a private message to any of the following moderators whose user IDs are listed in this post from time to time: <a href="https://www.lfgss.com/comments/7432589/">https://www.lfgss.com/comments/7432589/</a>.</p>

<p>When reporting content please provide the information described below in your notice to us:</p>

<ul>
	<li>your name and contact details; and</li>
	<li>a statement explaining whether you believe that the content you are contacting us about: (i) infringes your rights; (ii) the rights of a third party who you represent; (iii) is Illegal Content, or (iv) you otherwise believe the content breaches the Rules of Acceptable Use.</li>
</ul>

<p>We will take the action that we believe is appropriate depending on the nature of the content you report. This may include taking no action where we believe the reported content does not infringe any rights or the Rules of Acceptable Use. We are not obliged to discuss or inform you about our chosen course of action following a report from you.</p>

<h3>Advertisements on the forum</h3>

<p>We and our selected business partners may provide advertising to you through the forum or by other methods such as email. This advertising may be based on your User Content or other information available through the forum. When delivering advertising we will only use information that identifies you as set out in the forum privacy policy <a href="{{.LinkToPrivacyPolicy}}">{{.LinkToPrivacyPolicy}}</a>.</p>

<h3>Hyperlinks</h3>

<p>We may amend, re-format, modify or replace any website link you add to the forum. We may do this to improve the appearance of the link and/or to ensure the link used is that of one of our partners or affiliates. We may also direct users to an interstitial or warning page when we suspect the link is to content that may harm users or their devices such as links to malware or a shock site. We may also change how the link is displayed so that information from the link destination (such as content and media from the linked webpage) is displayed within the forum. For example, we may embed video or other media from a linked webpage into the forum.</p>

<h3>Fees</h3>

<p>The forum is free but some features, services or goods we may sell through the forum may require you to make a payment in order for you to receive or use them. You will be provided with information about any fees associated with the features, services or goods we provide before you purchase them and will be asked to approve any payment before it is taken from you.</p>

<p>Where you follow a link to other sites or resources and make a purchase from that site or resource the purchase is solely between you and the seller. Complaints, questions and claims related to the sale of the product or service should be directed to the seller.</p>

<p>We are not the seller of any product or service you purchase from third parties, including other users of this forum, we have no control and do not give any commitment relating to the existence, quality, genuineness of the product or service, the truth or accuracy of the picture or description of the product or service or any other content made available by the third party. Your use of the classifieds section of this forum and any other transactions you may carry out with users of this forum is entirely at your own risk.</p>

<p>You will be responsible for any fees that you incur by connecting to the forum such as Internet access costs or mobile phone data charges. If you are unsure whether you will incur such fees, we suggest that you log off the Internet and speak with your Internet service provider or mobile network operator.</p>

<h3>Ending our relationship</h3>

<p>If at any time you do not feel that you can agree to these Terms of Use or any changes made to the Terms of Use or the forum, you must immediately stop using the forum.</p>

<p>If you wish to end your use of the forum, please contact us at <a href="mailto:support@microcosm.cc">support@microcosm.cc</a> for deactivating your account.</p>

<p>We may immediately end your use of the forum if you break the Rules of Acceptable Use, any other important rule(s), or terms and conditions we set for accessing and using the forum including these Terms of Use.</p>

<p>We may end your use of the forum if your account has been inactive for 6 months.</p>

<p>We may also withdraw the forum as long as we give you reasonable notice that we plan to do so.</p>

<p>If you or we end your use of the forum or we withdraw the forum as described in this section, we may delete or modify your User Content, account or any other information we hold about you. You will also lose any rights you have to use the forum or access our content or your User Content. We will not offer you compensation for any losses.</p>

<h3>User Content Disclaimer</h3>

<p>Our forum makes available your User Content and User Content provided by other users of the forum. We do not produce such content and cannot be responsible for it in anyway.</p>

<h3>Our liability/responsibility to you</h3>

<p>This section sets out our entire responsibility to you under these Terms of Use.</p>

<p>Unfortunately, due to the nature of the Internet and technology, the forum is provided on an "as available" and "as is" basis. This means that we are unable to promise that your use of the forum will be uninterrupted, without delays, error-free or meet your expectations and we do not give any commitment relating to the performance or availability of the forum in these Terms of Use and, to the extent we are able to do so, we exclude any commitments that may be implied by law.</p>

<p>To the extent permitted by the law our total responsibility for any claims under these Terms of Use, related to the use of the forum or related to any other services or goods we may sell to you is limited to the amount you have paid us to provide the forum, services or goods (as the case may be) to you. For the avoidance of doubt, this does not include any funds which you have donated to or in support of us on a voluntary basis or otherwise for no consideration. In every case, you agree that we will not be liable for any loss or damage that is not reasonably foreseeable.</p>

<p>You will also not hold us responsible for any products or services you purchase from any other third party through the forum.</p>

<h3>Resolving disputes</h3>

<p>If you have a dispute with us relating to the forum, in the first instance please contact us at <a href="mailto:support@microcosm.cc">support@microcosm.cc</a> and attempt to resolve the dispute with us informally.</p>

<p>In the unlikely event that we are not able to resolve the dispute informally, we will discuss and agree with you the most effective way of resolving the dispute using mediation or arbitration based on the nature of our dispute.</p>

<h3>Changes to the forum</h3>

<p>We are constantly updating and improving the forum to try and find ways to provide you with new and innovative features and services. Improvements and updates are also made to reflect changing technologies, tastes, behaviours and the way people use the Internet and our forum. In order to do this, we may need to update, reset, stop offering and/or supporting a particular part of the forum or feature relating to the forum ("<strong>changes to the forum</strong>"). These changes to the forum may affect your past activities on the forum, features that you use and your User Content. Any changes to the forum could involve your User Content being deleted or reset.</p>

<p>You agree that a key characteristic of our forum is that changes to the forum will take place over time and this is an important basis on which we grant you access to the forum. Once we have made changes to the forum, your continued use of the forum will show that you have accepted any changes to the forum. You are always free to stop using the forum or deactivate your account by contacting us at <a href="mailto:support@microcosm.cc">support@microcosm.cc</a>.</p>

<p>We will try, where possible and reasonable, to contact you to let you know about any significant changes to the forum.</p>

<h3>Changes to the documents</h3>

<p>We may revise these Terms of Use from time to time but the current version will always be at <a href="/about/terms/">{{.CustomerURL}}/about/terms/</a>.</p>

<p>Changes will usually occur because of new features being added to the forum, changes in the law or where we need to clarify our position on something.</p>

<p>We will try, where possible and reasonable, to contact you to let you know about any significant changes to any of the documents referred to in these Terms of Use. We may contact you through the forum (for example by asking you to accept the changes before you continue to use the forum) or via a separate email.</p>

<p>Normally, we will try to give you some warning before the new terms become effective. However, sometimes changes will need to be made immediately and if this happens we will not give you any notice.</p>

<h3>Documents that apply to our relationship with you</h3>

<p>The current version of the Terms of Use contains the only terms and conditions that apply to your use of the forum. Older versions of the Terms of Use will no longer apply to our relationship and will be completely replaced by the current version.</p>

<p>We intend to rely on these Terms of Use as setting out the written terms of our agreement with you for the provision of the forum. If part of the Terms of Use cannot be enforced then the remainder of the Terms of Use will still apply to our relationship.</p>

<p>Where we have agreed with you that other terms and conditions will govern the other services or products we provide to you outside of the forum those terms and conditions will govern that relationship. If any other terms and conditions we have agreed with you contradict these Terms of Use in respect of your use of the forum, these Terms of Use will prevail.</p>

<p>If you do not comply with these Terms of Use and we do not take action immediately, this does not mean we have given up any right we have and we may still take action in the future.</p>

<h3>Law</h3>

<p>English law will apply to all disputes and the interpretation of these Terms of Use. The English courts will have non-exclusive jurisdiction over any dispute arising from or related to your use of the forum.</p>

<h3>Contact, feedback and complaints</h3>

<p>If you need to contact us in relation to these Terms of Use or any other document mentioned in them, please email us at <a href="mailto:support@microcosm.cc">support@microcosm.cc</a>.</p>

<p>We value hearing from our users and are always interested in learning about ways we can improve the forum. By providing your feedback you agree that you are giving up any rights you have in your feedback so that we may use and allow others to use it without any restriction and without any payment to you.</p>

<p>Last modified {{.TermsOfUseLastModified}}</p>

<h3>Appendix 1: Illegal content under the Online Safety Act 2023</h3>

<p>Priority illegal content</p>

<table>
	<tr>
		<th>No.</th>
		<th>Kind of illegal content</th>
		<th>Offences</th>
	</tr>
	<tr>
		<td>1.</td>
		<td>Terrorism</td>
		<td><ol type="a">
			<li>Offences related to information likely to be of use to a terrorist;</li>
			<li>Offences relating to training for terrorism;</li>
			<li>A series of offences relating to 'proscribed organisations';</li>
			<li>Other offences involving encouraging terrorism or disseminating terrorist materials;</li>
			<li>Miscellaneous, more specific terrorism offences; and</li>
			<li>Offences relating to financing terrorism.</li>
		</ol></td>
	</tr>
	<tr>
		<td>2.</td>
		<td>Child sexual exploitation and abuse</td>
		<td>Schedule 6 of the Online Safety Act 2023</td>
	</tr>
	<tr>
		<td>2A</td>
		<td>Grooming</td>
		<td>Schedule 6 of the Online Safety Act 2023</td>
	</tr>
	<tr>
		<td>2B</td>
		<td>CSAM</td>
		<td>Material relating to child sexual abuse (including images and URLs)</td>
	</tr>
	<tr>
		<td>3</td>
		<td>Hate</td>
		<td>
			Public Order Act 1986-
			<ol type="a">
				<li>section 18 (use of words or behaviour or display of written material);</li>
				<li>section 19 (publishing or distributing written material);</li>
				<li>section 21 (distributing, showing or playing a recording);</li>
				<li>section 29B (use of words or behaviour or display of written material);</li>
				<li>section 29C (publishing or distributing written material);</li>
				<li>section 29E (distributing, showing or playing a recording).</li>
			</ol>
			Crime and Disorder Act 1998-
			<ol type="a">
				<li>section 31 (racially or religiously aggravated public order offences);</li>
				<li>section 32 (racially or religiously aggravated harassment etc).</li>
			</ol>
		</td>
	</tr>
	<tr>
		<td>4.</td>
		<td>Harassment, stalking threats and abuse</td>
		<td><ol type="a">
			<li>Threats (including hate), encompassing:
				<ol type="i">
					<li>threatening behaviour which is likely to cause fear or alarm</li>
					<li>threatening behaviour which is likely to cause harassment or distress</li>
					<li>threats which are likely to stir up racial hatred</li>
					<li>threats which are likely to stir up hatred on the basis of religion or sexual orientation</li>
					<li>threats which may provoke violence</li>
				</ol>
			</li>
			<li>Abuse and insults (including hate), encompassing:
				<ol type="i">
					<li>abusive behaviour which is likely to cause fear or alarm</li>
					<li>abusive behaviour which is likely to cause harassment or distress</li>
					<li>abuse which is likely to stir up racial hatred</li>
					<li>abuse which may provoke violence</li>
				</ol>
			</li>
			<li>Other content likely to amount to harrassment </li>
		</ol></td>
	</tr>
	<tr>
		<td>5.</td>
		<td>Controlling or coercive behaviour</td>
		<td>Controlling or coercive behaviour in an intimate or family relationship (overlap with harassment).</td>
	</tr>
	<tr>
		<td>6.</td>
		<td>Intimate image abuse</td>
		<td>Offences related to non-consensual disclosure of intimate images</td>
	</tr>
	<tr>
		<td>7.</td>
		<td>Extreme pornography</td>
		<td>Possession of extreme pornographic images</td>
	</tr>
	<tr>
		<td>8.</td>
		<td>Sexual exploitation of adults</td>
		<td><ol type="a">
			<li>Causing or inciting prostitution for gain; and</li>
			<li>Controlling a prostitute for gain</li>
		</ol></td>
	</tr>
	<tr>
		<td>9.</td>
		<td>Human trafficking</td>
		<td><ol type="a">
			<li>the recruitment of another person,</li>
			<li>the transportation or transfer of another person,</li>
			<li>the harbouring or receiving of another person,</li>
			<li>the exchange or transfer of control over another person, or</li>
			<li>the arrangement or facilitation of any of these actions</li>
		</ol></td>
	</tr>
	<tr>
		<td>10.</td>
		<td>Unlawful immigration</td>
		<td><ol type="a">
			<li>Offences relating to illegal entry into the UK; and</li>
			<li>Assisting unlawful immigration.</li>
		</td>
	</tr>
	<tr>
		<td>11.</td>
		<td>Fraud and financial services</td>
		<td><ol type="a">
			<li>False claims to be authorised or exempt for the purposes of carrying on regulated activity (the first of the financial services offences);</li>
			<li>Fraud by false representation;</li>
			<li>Fraud by abuse of position and participating in fraudulent business carried on by sole trader etc.;</li>
			<li>Other financial services offences;</li>
			<li>Fraud related to misleading statements or impressions about investments;</li>
			<li>Offences related to articles for use in fraud; and</li>
			<li>Offences related to criminal property.</li>
		</ol></td>
	</tr>
	<tr>
		<td>12.</td>
		<td>Proceeds of crime</td>
		<td>Concealing etc criminal property); arrangements facilitating acquisition etc of criminal property; acquisition, use and possession of criminal property (overlap with fraud).</td>
	</tr>
	<tr>
		<td>13.</td>
		<td>Drugs and psychoactive substances</td>
		<td>Relating to relating to unlawful supplying, or offering to supply of controlled drugs, articles used for the administration or preparation of controlled drugs ('drugs articles'), and psychoactive substances.</td>
	</tr>
	<tr>
		<td>14</td>
		<td>Firearms and other weapons</td>
		<td><ol type="a">
			<li>Firearms, their parts, ammunition, including air weapons and shotguns;</li>
			<li>3D printing instructions for guns and gun parts;</li>
			<li>Knives;</li>
			<li>'Offensive' weapons, including axes and crossbows, knuckledusters, corrosive substances, blowpipes and truncheons; and</li>
			<li>Imitation firearms.</li>
		</ol></td>
	</tr>
	<tr>
		<td>15.</td>
		<td>Encouraging suicide or assisting suicide</td>
		<td>Encouraging or assisting suicide.</td>
	</tr>
	<tr>
		<td>16</td>
		<td>Foreign Interference</td>
		<td>S13 off National Security Act 2023 offence</td>
	</tr>
	<tr>
		<td>17</td>
		<td>Animal Welfare</td>
		<td>Animal Welfare Act 2006: unnecessary suffering of an animal</td>
	</tr>
</table>

<p>Non-priority illegal content</p>

<table>
	<tr>
		<th>No.</th>
		<th>Kind of illegal content</th>
		<th>Offences</th>
	</tr>
	<tr>
		<td>1.</td>
		<td>Epilepsy trolling</td>
		<td>Sending a flashing image with intention that it would be seen by a person with epilepsy or where it was reasonably foreseeable that this would be the case.</td>
	</tr>
	<tr>
		<td>2.</td>
		<td>Cyberflashing</td>
		<td>the offence of sending or giving a photograph or video of the genitals with the intent of causing alarm, distress or humiliation, or for the purpose of sexual gratification on the behalf of the sender.</td>
	</tr>
	<tr>
		<td>3.</td>
		<td>Self-harm</td>
		<td>Assisting or encouraging 'serious' acts of self-harm.</td>
	</tr>
	<tr>
		<td>4.</td>
		<td>False communications</td>
		<td>Sending a message which conveys knowingly false information with the intent of causing non-trivial psychological or physical harm to the likely audience</td>
	</tr>
	<tr>
		<td>5.</td>
		<td>Obscene content</td>
		<td>Sending or causing to be sent, online, a message that is of an obscene character (where the obscenity relates to torture or extreme cruelty).</td>
	</tr>
</table>
`
	return doc
}

var legalPrivacyPolicyLastModified, _ = time.Parse(time.DateOnly, "2014-03-06")

// GetRawPrivacyPolicyText returns the privacy policy
func GetRawPrivacyPolicyText() string {
	const doc string = `<h2>{{.CustomerName}}</h2>

<h2>Forum Privacy Policy</h2>

<h3>Our approach to privacy</h3>

<p>The forum <a href="{{.CustomerURL}}">{{.CustomerURL}}</a> (the "<strong>forum</strong>") is operated by the person, company or organisation detailed in the forum administrator section of the forum ("<strong>we</strong>" or "<strong>us</strong>"). We control how your personal information is used.</p>

<p>We are committed to protecting and respecting your privacy. This privacy policy sets out how we collect, use and share information that identifies you or is associated with you ("<strong>personal information</strong>") when you use our forum.</p>

<p>By setting up an account with us or using and accessing the forum, you consent to the use of your personal information as described in this privacy policy.</p>

<p>You should also be aware that we use cookies to store and access information whilst providing the forum. You can find out more about our use of cookies in our cookies policy <a href="{{.LinkToCookiePolicy}}">{{.LinkToCookiePolicy}}</a>.</p>

<h3>Personal information we hold about you</h3>

<p>We collect personal information from you when you voluntarily submit information directly to us or the forum. This can include information you provide when you register to use the forum, complete a form, comment on the forum, correspond with us, subscribe to our email lists or respond to surveys or promotions. We also collect personal information indirectly from you such as information about the pages you look at on the forum or the device you connect to the forum with. We may also collect personal information about you from third parties such as other users of the forum.</p>

<p>We describe the categories of personal information we collect in more detail below:</p>

<p><strong>Contact details</strong>: such as your name and email address.</p>

<p><strong>Profile information</strong>: such as user name, your interests and experiences and your profile picture.</p>

<p><strong>Comments and opinions</strong>: such as comments and opinions you leave on the forum or when you respond to our surveys, promotions or other user content such as classified ads, events or polls.</p>

<p><strong>Private messages to us</strong>: private messages you send to us that you do not make publicly available on the forum.</p>

<p><strong>Private messages to other users</strong>: private messages you send to other users that you do not make publicly available on the forum.</p>

<p><strong>Payment and transaction information</strong>: we will keep records of the products and services you purchase from us and the payment method you use for the transaction.</p>

<p><strong>Your preferences</strong>: choices you make such as notification preferences or choices about how the forum is displayed.</p>

<p><strong>Information about how you use and connect to the online service</strong></p>

<p>We collect information about how you use the forum such as the pages and links you access, the time you access the forum and duration you are on it, the website you come to the forum from or go to after leaving the forum and selections and choices you make when using the forum.</p>

<p>We also collect information about the computer or other electronic devices you use to connect to the forum such as details about the type of device (which can include unique device identifying numbers), its operating system, browser and applications connected to the forum through the device, your Internet service provider or mobile network, your IP address and your device's telephone number (if it has one).</p>

<p>Typically, the information we collect about how you use or connect to the forum is associated with your device and not you as a named individual. However, if you are logged into a registered account this information may also be associated with your account.</p>

<p><strong>Information about your location</strong></p>

<p>Other than information you choose to provide to us, we do not collect information about your precise location. Your device's IP address may help us determine an approximate location but this will be no more precise than the city, state or country you are using your device in.</p>

<p><strong>Information provided by other users</strong></p>

<p>Other users provide information that is associated with you such as comments and opinions they post on the forum that identify you or their responses to your comments and opinions or messages. For example, other users could provide information that is associated with you when responding to a post you have made on the forum or when addressing queries you have about classified ads or events.</p>

<p><strong>Information provided by other organisations</strong></p>

<p>Other organisations may provide information that we associate with you where they are lawfully permitted to share it, such as contact details, demographic data, or Internet navigation information.</p>

<p><strong>Information from fraud and crime prevention organisations</strong></p>

<p>If fraudulent or other criminal activity is suspected relating to your account, we may be provided with information by credit reference, fraud and crime prevention organisations.</p>

<h3>How we use your information</h3>

<p>We will also use your personal information for the purposes specifically described below:</p>

<p><strong>Contact details</strong>: we will use your contact details so that we can communicate with you directly about the service and about queries, issues or concerns you or other users have. We may also use your contact details to send you information about products and services that may be of interest to you (such communications will be in accordance with your marketing preferences).</p>

<p><strong>Profile information</strong>: we will use your profile information to create a public profile for you on the forum.</p>

<p><strong>Comments and opinions</strong>: we will display your public comments, opinions and responses on the public areas of the forum. We will not publicly display your private messages or private interactions with us or another user. We will also use your comments and opinions to determine topics, products and services that may be of interest to you.</p>

<p><strong>Payment and transaction information</strong>: we will use payment and transaction information you provide to process payment for any service or product you purchase from us. We may also use the history of your transactions with us to determine topics, products and services that may be of interest to you.</p>

<p><strong>Private messages to us</strong>: we will use private messages you send to us to help us respond to any query, issue or concern you raise.</p>

<p><strong>Private messages to other users</strong>: we will facilitate the delivery and storage of private messages between you and other users of the forum.</p>

<p><strong>Your preferences</strong>: we will use your preference settings for the purpose of providing notifications, displaying the forum and following other choices you are able to make about how the service is delivered.</p>

<p><strong>Information about how you use and connect to the online service</strong>: we use this information to present the forum to you on your device. We will also use this information to determine topics, products and services that may be of interest to you.</p>

<p><strong>Information about your location</strong>: we may use an approximate location to ensure content on the forum is relevant to the city, state or country you are using your device in. We will not collect or track your exact location without your consent.</p>

<p><strong>Information provided by other users</strong>: we will display comments and opinions posted by other users in the forum which may contain personal information about you. If you are concerned about any content which contains personal information about you please report it to us. We will use your personal information that is contained in private messages sent to us by other users to respond to any query, issue or concern they raise.</p>

<p><strong>Information about topics, products and services that may be of interest to you</strong>: we will use information we gather about topics, products and services that may be of interest to you to help us improve the forum and develop new products and services. We may also use this information to help us decide on marketing that may be relevant to you.</p>

<p><strong>Information about fraudulent or criminal activity relating to your account</strong>: we will use information about fraudulent or criminal activity relating to your account for the purposes of preventing fraud or crime.</p>

<p><strong>All personal information: will use all the personal information we collect to</strong>: (i) monitor and improve the forum and our procedures and processes; and (ii) to help us develop new products and services. Our use of your personal information in this way will not result in information that was not previously publicly available being made public on the forum. We will also not use information about fraudulent or criminal activity relating to your account in the development of new products and services.</p>

<p>We may anonymise and aggregate any of the information we collect (so that it does not directly identify you). We may use anonymised information for purposes that include testing our IT systems, research, data analysis, improving the forum and developing new products and features.</p>

<h3>Information we share with third parties</h3>

<p>We may share your personal information with the following parties:</p>

<p><strong>Companies in the same group of companies as us</strong>: our subsidiaries (i.e. any organisation we own or control) or our ultimate holding company (i.e. any organisation that owns or controls us) and any subsidiaries it owns. These companies will only use your personal information in the same way as we can under this privacy policy.</p>

<p><strong>Service providers and advisors</strong>: third parties who provide a service to us, for example, we use third party email provider to help us deliver your email notifications to you. These third parties will only be allowed to use your personal information in accordance with our instructions and will be required to keep your information secure.</p>

<p><strong>Other users of the forum</strong>: your comments and opinions, your user name, your profile picture, the description of yourself that you add to your profile and other user generated or supplied content will be available to other users of the forum. Some actions, such as declaring attendance of an event, may also grant other users of the forum access to your contact details.</p>

<p><strong>Purchasers of our business</strong>: personal information may be disclosed or transferred to buyers or perspective buyers of our business or any of our assets as part of the sale.</p>

<p><strong>Law enforcement, regulators and other parties for legal reasons</strong>: third parties who we are under a legal obligation to disclose your personal information to or who we need to disclose your personal information to protect our rights, property or safety or the rights, property or safety of others, detect and investigate illegal activities and breaches of any agreement we have with you.</p>

<p>We do not disclose information about identifiable individuals to anyone else except as set out above. We may provide third parties with aggregate statistical information and analytics about users of the forum but will we make sure no one can be identified from this information before we disclose it.</p>

<h3>Marketing and Advertising</h3>

<p><strong>Email Marketing</strong> - From time to time we may contact you by email with information about our products and services. If you do not want us to send you email marketing, please let us know by sending an email to us at <a href="mailto:{{.CustomerEmail}}">{{.CustomerEmail}}</a>. You can also unsubscribe from our marketing emails by clicking on the unsubscribe link (where available) in the emails we send to you.</p>

<p><strong>Email Notifications About Activity on the Forum</strong> - We will send you email notifications when certain actions take place on the forum. Such notifications will be sent in accordance with your notification settings under your user profile. If you want to stop or change the notifications you receive you can do so by amending your notification setting in your user profile.</p>

<p><strong>Advertising on the Forum and other Websites</strong> - We or our advertising partners may show advertisements to you on the forum or on other websites. To do this we or our advertising partners may collect information about how you use the forum or the types of other web pages, content and ads you, or others who are using the device you connect to the forum, interact with, visit or view. The information collected might also include the IP address of the device you connect to the forum.</p>

<p>We or our advertising partners use the information that is collected to make sure adverts you see either on the forum or other websites or apps (where our partners have advertising space) are more relevant to you.</p>

<p>The techniques our advertising partners use recognise the device you are using but are not aimed at you as a named individual. Typically cookies and similar technologies are used to provide this type of advertising. You can find out more about cookies and how to manage their use by reading our cookies policy <a href="{{.LinkToCookiePolicy}}">{{.LinkToCookiePolicy}}</a>.</p>

<h3>Storing and transferring your personal information</h3>

<p><strong>Timeframe</strong>: your personal information will be kept by us for as long as your account remains active. If you deactivate your account your personal information will be kept for a reasonable period of time after this so that we can complete any activities it is being used for. Your comments on the forum will be associated with your account as long as your account remains active. If you deactivate your account, your comments will be kept (as they form part of the discussions on the forum) but will no longer be associated with your account.</p>

<p><strong>Cookies and Local Storage</strong>: we may store information (which may include personal information) locally on your device using cookies and other browser and application storage methods. For further information please see our cookies policy <a href="{{.LinkToCookiePolicy}}">{{.LinkToCookiePolicy}}</a>.</p>

<p><strong>International Transfers of your Personal Information</strong>: the personal information we collect may be transferred to and stored in countries outside of the jurisdiction you are in. Some of these jurisdictions offer differing levels of protection in respect of personal information and may, in certain instances, be less protective than the jurisdiction you are typically resident in. We will take all reasonable steps to ensure that your personal information is treated securely and in accordance with this privacy policy.</p>

<h3>Accessing your personal information</h3>

<p>Most of the personal information we hold about you is available directly from viewing your account and looking back over your own activities. If you think there is other information you would like to access, please contact us using the contact details set out in the "Contacting us" section below. Please provide as much information as you can about the information you are looking for and we will be happy to try and help you find it.</p>

<h3>Links to third party sites</h3>

<p>The forum may, from time to time, contain links to and from third party websites of our partner networks, advertisers, partner merchants, retailers and affiliates. If you follow a link to any of these websites, please note that these websites have their own privacy policies and that we do not accept any responsibility or liability for their policies. Please check the individual policies before you submit any information to those websites.</p>

<h3>Our policy towards children</h3>

<p>Our forum is only available to persons aged 13 or over.</p>

<p>The forum is not directed to persons under 13 and we do not knowingly collect personal information from children under 13. If you become aware that your child has provided us with personal information, without your consent, then please contact us using the details below so that we can take steps to remove such information and terminate the child's account.</p>

<h3>Changes to this policy</h3>

<p>We may update this privacy policy from time to time and so you should review this policy periodically. When we change this privacy policy in a material way, we will update the "last modified" date at the bottom of this privacy policy. Changes to this privacy policy are effective when they are posted on this page.</p>

<h3>Notice to you</h3>

<p>If we need to provide you with information about something, whether for legal, marketing or other business related purposes, we will select what we believe is the best way to get in contact with you. We will usually do this through email or by placing a notice on the forum. The fact that we may send notices to you will not stop you from being able to opt out of certain types of contact as described in this privacy policy.</p>

<h3>Contacting us</h3>

<p>Questions, comments and requests regarding this privacy policy are welcome and should be addressed to <a href="mailto:{{.CustomerEmail}}">{{.CustomerEmail}}</a>.</p>

<p>This privacy policy was last modified on {{.PrivacyPolicyLastModified}}.</p>`

	return doc
}

var legalCookiePolicyLastModified, _ = time.Parse(time.DateOnly, "2014-03-06")

// GetRawCookiePolicyText returns the cookie policy
func GetRawCookiePolicyText() string {
	const doc string = `<h2>{{.CustomerName}}</h2>

<h2>Forum Cookie Policy</h2>

<h3>Summary</h3>

<p>The forum <a href="{{.CustomerURL}}">{{.CustomerURL}}</a> (the "<strong>forum</strong>") is operated by the person, company or organisation detailed in the forum administrator section of the forum ("<strong>we</strong>" or "<strong>us</strong>").</p>

<p>This cookie policy explains what cookies are and how we use them on the forum.  We encourage you to read the policy in full so that you can understand what information we collect using cookies and how that information is used.</p>

<p><em>By using our forum you are agreeing that we can use cookies in accordance with this policy.</em></p>

<h3>What are cookies?</h3>

<p>Cookies are text files, containing small amounts of information, which are downloaded to your browsing device (such as a computer or smartphone) when you visit a website. Cookies can be recognised by the website that downloaded them - or other websites that use the same cookies.  This helps websites know if the browsing device has visited them before.</p>

<h3>What are cookies used for?</h3>

<p>Cookies do lots of different jobs, like helping us understand how our forum is being used, letting you navigate between pages efficiently, remembering your preferences, and generally improving your browsing experience. Cookies can also help ensure marketing you see online is more relevant to you and your interests.</p>

<h3>What types of cookies are used by {{.CustomerName}} on the forum?</h3>

<p>The types of cookies used on our forum can generally be put into one of the following categories: strictly necessary; analytics; and functionality. You can find out more about each of the cookie categories in the table below.</p>

<table>
	<tr>
		<th>Strictly Necessary Cookies</th>
		<td>These cookies are essential to make the forum work. They enable you to move around the forum and use its features. Without these cookies, services that are necessary for you to be able to use our forum such as accessing secure areas cannot be provided.</td>
	</tr>
	<tr>
		<th>Analytics Cookies</th>
		<td>These cookies collect information about how people are using our forum, for example which pages are visited the most often, how people are moving from one link to another and if they get error messages from certain pages. These cookies don't gather information that identifies you.  All information these cookies collect is grouped together with information from other people's use of our forum on an anonymous basis. Overall, these cookies provide us with analytical information about how our forum is performing and how we can improve it.</td>
	</tr>
	<tr>
		<th>Functionality Cookies</th>
		<td>These cookies allow us to remember choices you make and tailor our forum to provide enhanced features and content to you. For example, these cookies can be used to remember your user name, language choice or country selection, they can also be used to remember changes you've made to text size, font and other parts of pages that you can customise.</td>
	</tr>
	<tr>
		<th>Advertising Cookies</th>
		<td>These cookies are used to deliver advertisements that are more relevant to you and your interests. They are also used to limit the number of times you see an advertisement as well as help measure the effectiveness of the advertising campaign. They remember that you have visited a website and this information may be shared with other organisations such as advertising partners. This means after you have been to our site you may see some advertisements about our services elsewhere on the Internet. Our cookies recognise the device you are using but are not aimed at you as a named individual.</td>
	</tr>
</table>

<h3>How long will cookies stay on my browsing Device?</h3>

<p>The length of time a cookie will stay on your browsing device depends on whether it is a "persistent" or "session" cookie.  Session cookies will only stay on your device until you stop browsing.  Persistent cookies stay on your browsing device after you have finished browsing until they expire or are deleted.</p>

<h3>First and third-party cookies</h3>

<p>"First party cookies" are cookies that belong to us and that we place on your device. "Third-party cookies" are cookies that another party places on your browsing device when you visit our site.</p>

<p>Third parties setting cookies from our forum will be providing a service to us or a function of the forum.</p>

<h3>How to manage cookies from this website</h3>

<p>You can usually use the browser that you are viewing this forum through to enable, disable or delete cookies. To do this, follow the instructions provided by your browser (usually located within the “Help”, “Tools” or “Edit” settings). Please note that if you set your browser to disable cookies, you may not be able to access secure areas of the website and other parts of the website may also not work properly.</p>

<p>You can find out more information about how to change your browser cookie settings at <a href="http://www.allaboutcookies.org">http://www.allaboutcookies.org</a>.</p>

<h3>Changes</h3>

<p>Information about the cookies used by us may be updated from time to time, so please check back on a regular basis for any changes.</p>

<p>The last modification date of this document is shown at the bottom of this page.</p>

<h3>Questions</h3>

<p>If you have any questions about this cookie policy, please contact us by email at <a href="mailto:{{.CustomerEmail}}">{{.CustomerEmail}}</a>.</p>

<p>Last modified {{.CookiePolicyLastModified}}</p>
`

	return doc
}
